package adkagents

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// EnhancedModelProviderImpl 增强的模型提供者实现
type EnhancedModelProviderImpl struct {
	config           *config.Config
	apiKeyRepo       repository.APIKeyRepository
	apiKeyService    APIKeyService
	taskUsageService TaskUsageService
}

// NewEnhancedModelProvider 创建增强的模型提供者
func NewEnhancedModelProvider(
	cfg *config.Config,
	apiKeyRepo repository.APIKeyRepository,
	apiKeyService APIKeyService,
	taskUsageService TaskUsageService,
) (*EnhancedModelProviderImpl, error) {
	provider := &EnhancedModelProviderImpl{
		config:           cfg,
		apiKeyRepo:       apiKeyRepo,
		apiKeyService:    apiKeyService,
		taskUsageService: taskUsageService,
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
		// 数据库无可用模型，返回错误
		klog.Errorf("EnhancedModelProvider.GetModel: no available model in database")
		return nil, ErrNoAvailableModel
	}

	// 从数据库获取 API Key 配置
	apiKey, err := p.apiKeyRepo.GetByName(context.Background(), name)
	if err != nil {
		klog.Warningf("EnhancedModelProvider.GetModel: failed to get API Key %s: %v", name, err)
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

	klog.V(6).Infof("EnhancedModelProvider.GetModel: created model %s", name)
	return &chatModel.ChatModel, nil
}

// DefaultModel 获取默认模型
// 不再提供默认模型，直接从数据库获取最高优先级的模型
func (p *EnhancedModelProviderImpl) DefaultModel() *openai.ChatModel {
	apiKey, err := p.apiKeyRepo.GetHighestPriority(context.Background())
	if err != nil || apiKey == nil {
		klog.Errorf("EnhancedModelProvider.DefaultModel: no available model in database")
		return nil
	}
	chatModel, err := p.GetModel(apiKey.Name)
	if err != nil {
		klog.Errorf("EnhancedModelProvider.DefaultModel: failed to get model %s: %v", apiKey.Name, err)
		return nil
	}
	return chatModel
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

		// 创建 ChatModel 实例
		chatModel, err := p.createChatModel(apiKey)
		if err != nil {
			klog.Errorf("EnhancedModelProvider.GetModelPool: failed to create ChatModel for %s: %v", apiKey.Name, err)
			continue
		}

		models = append(models, chatModel)
	}

	klog.V(6).Infof("EnhancedModelProvider.GetModelPool: got %d available models", len(models))
	return models, nil
}

// createChatModel 创建 ChatModel 实例
func (p *EnhancedModelProviderImpl) createChatModel(apiKey *model.APIKey) (*ModelWithMetadata, error) {
	openaiConfig := &openai.ChatModelConfig{
		BaseURL:    apiKey.BaseURL,
		APIKey:     apiKey.APIKey,
		Model:      apiKey.Model,
		HTTPClient: NewClaudeHTTPClient(),
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
		LLMModel:   apiKey.Model,
	}, nil
}

// MarkModelUnavailable 标记模型为不可用
func (p *EnhancedModelProviderImpl) MarkModelUnavailable(ctx context.Context, modelName string, resetTime time.Time) error {
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
