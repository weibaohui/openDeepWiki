package adkagents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// EnhancedModelProviderImpl 增强的模型提供者实现
type EnhancedModelProviderImpl struct {
	config          *config.Config
	apiKeyRepo      repository.APIKeyRepository
	apiKeyService   APIKeyService
	defaultModel    *ModelWithMetadata
	modelCache      map[string]*ModelWithMetadata
	modelCacheMutex sync.RWMutex
	switcher        *ModelSwitcher
}

// NewEnhancedModelProvider 创建增强的模型提供者
func NewEnhancedModelProvider(
	cfg *config.Config,
	apiKeyRepo repository.APIKeyRepository,
	apiKeyService APIKeyService,
) (*EnhancedModelProviderImpl, error) {
	// 创建默认模型
	defaultChatModel, err := NewLLMChatModel(cfg)
	if err != nil {
		return nil, err
	}

	provider := &EnhancedModelProviderImpl{
		config:        cfg,
		apiKeyRepo:    apiKeyRepo,
		apiKeyService: apiKeyService,
		defaultModel: &ModelWithMetadata{
			ChatModel:  *defaultChatModel,
			APIKeyName: "default",
			APIKeyID:   0,
		},
		modelCache: make(map[string]*ModelWithMetadata),
		switcher:   NewModelSwitcher(apiKeyService),
	}

	return provider, nil
}

// GetModel 获取指定名称的模型
func (p *EnhancedModelProviderImpl) GetModel(name string) (*openai.ChatModel, error) {
	klog.V(6).Infof("EnhancedModelProvider.GetModel: name=%s", name)

	// 如果 name 为空，尝试使用数据库中的最高优先级模型
	if name == "" {
		apiKey, err := p.apiKeyRepo.GetHighestPriority(context.Background())
		if err == nil && apiKey != nil {
			klog.V(6).Infof("EnhancedModelProvider.GetModel: found highest priority model in DB: %s", apiKey.Name)
			// 使用查到的名称递归调用
			return p.GetModel(apiKey.Name)
		}
		// 数据库无可用模型，使用默认模型（Env配置）
		klog.V(6).Infof("EnhancedModelProvider.GetModel: using default model (Env fallback)")
		return &p.defaultModel.ChatModel, nil
	}

	// 检查缓存
	p.modelCacheMutex.RLock()
	if cachedModel, exists := p.modelCache[name]; exists {
		p.modelCacheMutex.RUnlock()
		klog.V(6).Infof("EnhancedModelProvider.GetModel: using cached model %s", name)
		return &cachedModel.ChatModel, nil
	}
	p.modelCacheMutex.RUnlock()

	// 从数据库获取 API Key 配置
	apiKey, err := p.apiKeyRepo.GetByName(context.Background(), name)
	if err != nil {
		klog.Warningf("EnhancedModelProvider.GetModel: failed to get API Key %s: %v, trying fallback", name, err)
		// 如果指定了名称但找不到，且该名称不是默认模型，尝试回退到默认模型?
		// 需求是：如果数据库没有模型，最后使用env环境变量中的模型兜底。
		// 但如果用户明确指定了 "gpt-4" 而数据库没有，是否应该 fallback 到 "env-model"?
		// 通常明确指定名字时不应该 fallback，否则会产生意外行为。
		// 但考虑到需求描述："如果数据库没有模型，最后使用env环境变量中的模型兜底"
		// 这可能主要针对自动选择场景。
		// 如果这里返回 error，调用方可能会失败。
		// 保持原逻辑：指定名字找不到则报错。
		return nil, ErrAPIKeyNotFound
	}

	// 检查是否可用
	if !apiKey.IsAvailable() {
		klog.Warningf("EnhancedModelProvider.GetModel: API Key %s is not available (status=%s, rate_limit_reset_at=%v)",
			name, apiKey.Status, apiKey.RateLimitResetAt)
		return nil, ErrModelUnavailable
	}

	// 创建 ChatModel 实例
	chatModel, err := p.createChatModel(apiKey)
	if err != nil {
		klog.Errorf("EnhancedModelProvider.GetModel: failed to create ChatModel for %s: %v", name, err)
		return nil, err
	}

	// 缓存模型
	p.modelCacheMutex.Lock()
	p.modelCache[name] = chatModel
	p.modelCacheMutex.Unlock()

	klog.V(6).Infof("EnhancedModelProvider.GetModel: created and cached model %s", name)
	return &chatModel.ChatModel, nil
}

// DefaultModel 获取默认模型
func (p *EnhancedModelProviderImpl) DefaultModel() *openai.ChatModel {
	return &p.defaultModel.ChatModel
}

// GetModelPool 获取模型池（按优先级排序）
func (p *EnhancedModelProviderImpl) GetModelPool(ctx context.Context, names []string) ([]*ModelWithMetadata, error) {
	klog.V(6).Infof("EnhancedModelProvider.GetModelPool: getting models for names %v", names)

	// 从数据库获取 API Key 配置列表
	var apiKeys []*model.APIKey
	var err error
	if len(names) == 0 {
		// 如果未指定名称，获取所有配置
		apiKeys, err = p.apiKeyRepo.List(ctx)
	} else {
		apiKeys, err = p.apiKeyRepo.ListByNames(ctx, names)
	}
	if err != nil {
		klog.Errorf("EnhancedModelProvider.GetModelPool: failed to get API Keys: %v", err)
		return nil, err
	}

	// 过滤可用的配置并创建模型
	models := make([]*ModelWithMetadata, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		if !apiKey.IsAvailable() {
			klog.V(6).Infof("EnhancedModelProvider.GetModelPool: skipping unavailable model %s", apiKey.Name)
			continue
		}

		// 检查缓存
		p.modelCacheMutex.RLock()
		if cachedModel, exists := p.modelCache[apiKey.Name]; exists {
			p.modelCacheMutex.RUnlock()
			models = append(models, cachedModel)
			continue
		}
		p.modelCacheMutex.RUnlock()

		// 创建 ChatModel 实例
		chatModel, err := p.createChatModel(apiKey)
		if err != nil {
			klog.Errorf("EnhancedModelProvider.GetModelPool: failed to create ChatModel for %s: %v", apiKey.Name, err)
			continue
		}

		// 缓存模型
		p.modelCacheMutex.Lock()
		p.modelCache[apiKey.Name] = chatModel
		p.modelCacheMutex.Unlock()

		models = append(models, chatModel)
	}

	klog.V(6).Infof("EnhancedModelProvider.GetModelPool: got %d available models", len(models))
	return models, nil
}

// createChatModel 创建 ChatModel 实例
func (p *EnhancedModelProviderImpl) createChatModel(apiKey *model.APIKey) (*ModelWithMetadata, error) {
	openaiConfig := &openai.ChatModelConfig{
		BaseURL:   apiKey.BaseURL,
		APIKey:    apiKey.APIKey,
		Model:     apiKey.Model,
		MaxTokens: &p.config.LLM.MaxTokens,
	}

	chatModel, err := openai.NewChatModel(context.Background(), openaiConfig)
	if err != nil {
		return nil, err
	}

	// 包装模型，添加 API Key ID 以便跟踪
	return &ModelWithMetadata{
		ChatModel:  *chatModel,
		APIKeyName: apiKey.Name,
		APIKeyID:   apiKey.ID,
	}, nil
}

// IsRateLimitError 判断错误是否为 Rate Limit 错误
func (p *EnhancedModelProviderImpl) IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	errMsg = strings.ToLower(errMsg)
	// 检查 HTTP 状态码
	if strings.Contains(errMsg, "429") {
		return true
	}

	// 检查错误消息
	rateLimitKeywords := []string{
		"rate limit",
		"quota exceeded",
		"too many requests",
		"rate-limited",
		"request rate exceeded",
		"请求次数超过限制",
		"超过限制",
		"每分钟请求次数",
	}

	lowerMsg := strings.ToLower(errMsg)
	for _, keyword := range rateLimitKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}

	return false
}

// MarkModelUnavailable 标记模型为不可用
func (p *EnhancedModelProviderImpl) MarkModelUnavailable(modelName string, resetTime time.Time) error {
	ctx := context.Background()

	// 获取 API Key 配置
	apiKey, err := p.apiKeyRepo.GetByName(ctx, modelName)
	if err != nil {
		return err
	}

	// 标记为不可用
	err = p.apiKeyService.MarkUnavailable(ctx, apiKey.ID, resetTime)
	if err != nil {
		return err
	}

	// 清除缓存
	p.modelCacheMutex.Lock()
	delete(p.modelCache, modelName)
	p.modelCacheMutex.Unlock()

	klog.Warningf("EnhancedModelProvider.MarkModelUnavailable: marked model %s as unavailable, reset at %v", modelName, resetTime)
	return nil
}

// GetNextModel 获取下一个可用模型
func (p *EnhancedModelProviderImpl) GetNextModel(ctx context.Context, currentModelName string, poolNames []string) (*ModelWithMetadata, error) {
	// 获取模型池
	models, err := p.GetModelPool(ctx, poolNames)
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, ErrNoAvailableModel
	}

	// 找到当前模型的位置
	currentIndex := -1
	for i, model := range models {
		if model.APIKeyName == currentModelName {
			currentIndex = i
			break
		}
	}

	// 如果当前模型不在池中，返回第一个可用模型
	if currentIndex == -1 {
		klog.V(6).Infof("EnhancedModelProvider.GetNextModel: current model not in pool, returning first model")
		return models[0], nil
	}

	// 返回下一个模型
	if currentIndex+1 < len(models) {
		nextModel := models[currentIndex+1]
		klog.V(6).Infof("EnhancedModelProvider.GetNextModel: switching from index %d to %d", currentIndex, currentIndex+1)
		return nextModel, nil
	}

	// 没有下一个模型
	return nil, ErrNoAvailableModel
}

// ErrAPIKeyNotFound API Key 不存在错误
var ErrAPIKeyNotFound = fmt.Errorf("api key not found")

// ErrModelUnavailable 模型不可用错误
var ErrModelUnavailable = fmt.Errorf("model unavailable")

// ErrNoAvailableModel 没有可用模型错误
var ErrNoAvailableModel = fmt.Errorf("no available model")
