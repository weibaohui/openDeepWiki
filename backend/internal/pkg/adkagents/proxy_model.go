package adkagents

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// ProxyChatModel 动态代理模型，支持自动切换
type ProxyChatModel struct {
	provider   *EnhancedModelProviderImpl
	modelNames []string

	tools   []*schema.ToolInfo
	toolsMu sync.RWMutex
}

// NewProxyChatModel 创建代理模型
func NewProxyChatModel(provider *EnhancedModelProviderImpl, modelNames []string) *ProxyChatModel {
	return &ProxyChatModel{
		provider:   provider,
		modelNames: modelNames,
	}
}

// Generate 实现 model.ChatModel 接口
func (p *ProxyChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 获取模型池
	models, err := p.provider.GetModelPool(ctx, p.modelNames)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		// 如果没有可用模型，且未指定特定模型名称（全局自动兜底模式），尝试使用 Env 默认模型
		if len(p.modelNames) == 0 {
			klog.Warningf("ProxyChatModel.Generate: no DB models available, falling back to Env default model")
			defaultModel := p.provider.DefaultModel()
			// 绑定工具
			p.toolsMu.RLock()
			tools := p.tools
			p.toolsMu.RUnlock()
			if len(tools) > 0 {
				if binder, ok := interface{}(defaultModel).(interface {
					BindTools(tools []*schema.ToolInfo) error
				}); ok {
					_ = binder.BindTools(tools)
				}
			}
			return defaultModel.Generate(ctx, input, opts...)
		}
		return nil, ErrNoAvailableModel
	}

	// 尝试使用第一个可用模型
	currentModel := models[0]
	klog.V(6).Infof("ProxyChatModel.Generate: using model %s (ID: %d)", currentModel.APIKeyName, currentModel.APIKeyID)

	// 绑定工具
	p.toolsMu.RLock()
	tools := p.tools
	p.toolsMu.RUnlock()

	if len(tools) > 0 {
		// 尝试绑定工具
		// 注意：我们需要确保 currentModel.ChatModel 实现了 BindTools
		// 这里使用反射或类型断言来检查
		if binder, ok := interface{}(&currentModel.ChatModel).(interface {
			BindTools(tools []*schema.ToolInfo) error
		}); ok {
			if err := binder.BindTools(tools); err != nil {
				klog.Warningf("ProxyChatModel.Generate: failed to bind tools to model %s: %v", currentModel.APIKeyName, err)
			}
		} else {
			klog.V(6).Infof("ProxyChatModel.Generate: model %s does not support BindTools", currentModel.APIKeyName)
		}
	}

	// 执行生成
	result, err := currentModel.ChatModel.Generate(ctx, input, opts...)
	if err != nil {
		// 检查 Rate Limit
		if p.provider.IsRateLimitError(err) {
			klog.Warningf("ProxyChatModel.Generate: rate limit hit for model %s: %v", currentModel.APIKeyName, err)

			// 解析重置时间
			resetTime := p.parseResetTime(err)
			if resetTime.IsZero() {
				// 默认 2 分钟
				resetTime = time.Now().Add(2 * time.Minute)
			}

			// 标记不可用
			if markErr := p.provider.MarkModelUnavailable(currentModel.APIKeyName, resetTime); markErr != nil {
				klog.Errorf("ProxyChatModel.Generate: failed to mark model unavailable: %v", markErr)
			}

			// 返回错误，让外部重试
			return nil, err
		}
		return nil, err
	}

	if result != nil && result.ResponseMeta != nil && result.ResponseMeta.Usage != nil {
		taskID, ok := ctx.Value("taskID").(uint)
		if !ok {
			klog.Infof("任务用量记录失败：未在上下文中获取到 taskID")
		} else if p.provider.taskUsageService != nil {
			if err := p.provider.taskUsageService.RecordUsage(ctx, taskID, currentModel.LLMModel, result.ResponseMeta.Usage); err != nil {
				klog.Infof("任务用量记录失败：taskID=%d, 模型=%s, err=%v", taskID, currentModel.LLMModel, err)
			}
		}
		klog.V(6).Infof("模型返回用量：model=%s, usage=%v", currentModel.LLMModel, result.ResponseMeta.Usage)
	}
	// 记录请求
	if currentModel.APIKeyID > 0 {
		_ = p.provider.apiKeyService.RecordRequest(ctx, currentModel.APIKeyID, true)
	}

	return result, nil
}

// Stream 实现 model.ChatModel 接口
func (p *ProxyChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// 获取模型池
	models, err := p.provider.GetModelPool(ctx, p.modelNames)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		// 如果没有可用模型，且未指定特定模型名称（全局自动兜底模式），尝试使用 Env 默认模型
		if len(p.modelNames) == 0 {
			klog.Warningf("ProxyChatModel.Stream: no DB models available, falling back to Env default model")
			defaultModel := p.provider.DefaultModel()
			// 绑定工具
			p.toolsMu.RLock()
			tools := p.tools
			p.toolsMu.RUnlock()
			if len(tools) > 0 {
				if binder, ok := interface{}(defaultModel).(interface {
					BindTools(tools []*schema.ToolInfo) error
				}); ok {
					_ = binder.BindTools(tools)
				}
			}
			return defaultModel.Stream(ctx, input, opts...)
		}
		return nil, ErrNoAvailableModel
	}

	// 尝试使用第一个可用模型
	currentModel := models[0]
	klog.V(6).Infof("ProxyChatModel.Stream: using model %s (ID: %d)", currentModel.APIKeyName, currentModel.APIKeyID)

	// 绑定工具
	p.toolsMu.RLock()
	tools := p.tools
	p.toolsMu.RUnlock()

	if len(tools) > 0 {
		if binder, ok := interface{}(&currentModel.ChatModel).(interface {
			BindTools(tools []*schema.ToolInfo) error
		}); ok {
			if err := binder.BindTools(tools); err != nil {
				klog.Warningf("ProxyChatModel.Stream: failed to bind tools to model %s: %v", currentModel.APIKeyName, err)
			}
		}
	}

	// 执行生成
	result, err := currentModel.ChatModel.Stream(ctx, input, opts...)
	if err != nil {
		// 检查 Rate Limit
		if p.provider.IsRateLimitError(err) {
			klog.Warningf("ProxyChatModel.Stream: rate limit hit for model %s: %v", currentModel.APIKeyName, err)

			// 解析重置时间
			resetTime := p.parseResetTime(err)
			if resetTime.IsZero() {
				resetTime = time.Now().Add(time.Hour)
			}

			// 标记不可用
			if markErr := p.provider.MarkModelUnavailable(currentModel.APIKeyName, resetTime); markErr != nil {
				klog.Errorf("ProxyChatModel.Stream: failed to mark model unavailable: %v", markErr)
			}

			return nil, err
		}
		return nil, err
	}

	// 记录请求
	if currentModel.APIKeyID > 0 {
		_ = p.provider.apiKeyService.RecordRequest(ctx, currentModel.APIKeyID, true)
	}

	return result, nil
}

// BindTools 实现 model.ChatModel 接口 (或 ToolBindable)
func (p *ProxyChatModel) BindTools(tools []*schema.ToolInfo) error {
	p.toolsMu.Lock()
	defer p.toolsMu.Unlock()
	p.tools = tools
	return nil
}

// WithTools 适配 model.ToolCallingChatModel 接口
// 注意：如果接口签名不匹配，Linter 会提示
func (p *ProxyChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	p.BindTools(tools)
	return p, nil
}

// parseResetTime 从错误中解析重置时间
func (p *ProxyChatModel) parseResetTime(err error) time.Time {
	if err == nil {
		return time.Time{}
	}

	errMsg := err.Error()

	// 尝试从错误消息中解析重置时间
	patterns := []struct {
		pattern  string
		duration time.Duration
	}{
		{`Try again in (\d+)s`, time.Second},
		{`Retry after (\d+)s`, time.Second},
		{`Try again in (\d+)m`, time.Minute},
		{`Retry after (\d+)m`, time.Minute},
		{`Try again in (\d+)h`, time.Hour},
		{`Retry after (\d+)h`, time.Hour},
	}

	for _, pt := range patterns {
		re := regexp.MustCompile(pt.pattern)
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) >= 2 {
			var duration int
			if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
				return time.Now().Add(time.Duration(duration) * pt.duration)
			}
		}
	}

	// 尝试解析具体时间
	timePatterns := []string{
		`Reset at (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`,
		`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`,
	}

	for _, pattern := range timePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) >= 2 {
			var resetTime time.Time
			if err := resetTime.UnmarshalText([]byte(matches[1])); err == nil {
				return resetTime
			}
		}
	}

	return time.Time{}
}
