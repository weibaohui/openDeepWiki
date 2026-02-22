package adkagents

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// ProxyChatModel 动态代理模型，支持自动切换
type ProxyChatModel struct {
	provider    *EnhancedModelProviderImpl
	modelNames  []string
	toolBinder  *ToolBinder
	rateLimiter *RateLimiter
}

// NewProxyChatModel 创建代理模型
func NewProxyChatModel(provider *EnhancedModelProviderImpl, modelNames []string) *ProxyChatModel {
	return &ProxyChatModel{
		provider:    provider,
		modelNames:  modelNames,
		toolBinder:  NewToolBinder(),
		rateLimiter: NewRateLimiter(provider),
	}
}

// Generate 实现 model.ChatModel 接口
func (p *ProxyChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	result, err := p.executeWithModel(ctx, input, opts, func(model *ModelWithMetadata) (any, error) {
		return model.ChatModel.Generate(ctx, input, opts...)
	})
	if err != nil {
		return nil, err
	}
	msg, ok := result.(*schema.Message)
	if !ok {
		klog.Errorf("ProxyChatModel.Generate: unexpected result type %T", result)
		return nil, fmt.Errorf("unexpected result type: expected *schema.Message, got %T", result)
	}
	return msg, nil
}

// Stream 实现 model.ChatModel 接口
func (p *ProxyChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	result, err := p.executeWithModel(ctx, input, opts, func(model *ModelWithMetadata) (any, error) {
		return model.ChatModel.Stream(ctx, input, opts...)
	})
	if err != nil {
		return nil, err
	}
	stream, ok := result.(*schema.StreamReader[*schema.Message])
	if !ok {
		klog.Errorf("ProxyChatModel.Stream: unexpected result type %T", result)
		return nil, fmt.Errorf("unexpected result type: expected *schema.StreamReader[*schema.Message], got %T", result)
	}
	return stream, nil
}

// executeWithModel 模板方法：消除 Generate 和 Stream 的重复代码
func (p *ProxyChatModel) executeWithModel(
	ctx context.Context,
	_ []*schema.Message,
	_ []model.Option,
	executor func(model *ModelWithMetadata) (any, error),
) (any, error) {
	// 1. 获取模型
	model, err := p.getModel(ctx)
	if err != nil {
		return nil, err
	}

	klog.V(6).Infof("ProxyChatModel: using model %s (ID: %d)", model.APIKeyName, model.APIKeyID)

	// 2. 绑定工具
	p.toolBinder.BindToModel(&model.ChatModel)

	// 3. 执行请求
	result, err := executor(model)
	if err != nil {
		// 4. 处理 Rate Limit 错误
		if p.rateLimiter.IsRateLimitError(err) {
			return nil, p.rateLimiter.HandleRateLimit(ctx, model.APIKeyName, err, 2*time.Minute)
		}
		return nil, err
	}

	// 5. 记录用量（仅 Generate）
	if msg, ok := result.(*schema.Message); ok && msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
		p.recordUsage(ctx, model.LLMModel, msg.ResponseMeta.Usage)
	}

	// 6. 记录请求
	if model.APIKeyID > 0 {
		if err := p.provider.apiKeyService.RecordRequest(ctx, model.APIKeyID, true); err != nil {
			klog.Warningf("ProxyChatModel: failed to record request for APIKeyID %d: %v", model.APIKeyID, err)
		}
	}

	return result, nil
}

// getModel 获取模型
func (p *ProxyChatModel) getModel(ctx context.Context) (*ModelWithMetadata, error) {
	models, err := p.provider.GetModelPool(ctx, p.modelNames)
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		klog.Errorf("ProxyChatModel: no available models in database")
		return nil, ErrNoAvailableModel
	}

	return models[0], nil
}

// BindTools 实现 model.ChatModel 接口
func (p *ProxyChatModel) BindTools(tools []*schema.ToolInfo) error {
	return p.toolBinder.BindTools(tools)
}

// WithTools 适配 model.ToolCallingChatModel 接口
func (p *ProxyChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	p.BindTools(tools)
	return p, nil
}

// recordUsage 记录用量
func (p *ProxyChatModel) recordUsage(ctx context.Context, modelName string, usage *schema.TokenUsage) {
	taskID, ok := ctx.Value("taskID").(uint)
	if !ok {
		klog.Infof("任务用量记录失败：未在上下文中获取到 taskID")
		return
	}

	if p.provider.taskUsageService != nil {
		if err := p.provider.taskUsageService.RecordUsage(ctx, taskID, modelName, usage); err != nil {
			klog.Infof("任务用量记录失败：taskID=%d, 模型=%s, err=%v", taskID, modelName, err)
		}
	}

	klog.V(6).Infof("模型返回用量：model=%s, usage=%v", modelName, usage)
}
