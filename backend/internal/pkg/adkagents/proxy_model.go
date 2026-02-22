package adkagents

import (
	"context"
	"fmt"
	"strings"
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
	result, err := p.executeWithRetry(ctx, input, opts, func(model *ModelWithMetadata) (any, error) {
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
	result, err := p.executeWithRetry(ctx, input, opts, func(model *ModelWithMetadata) (any, error) {
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

// executeWithRetry 执行请求并支持模型切换重试
func (p *ProxyChatModel) executeWithRetry(
	ctx context.Context,
	input []*schema.Message,
	opts []model.Option,
	executor func(model *ModelWithMetadata) (any, error),
) (any, error) {
	const maxAttempts = 3

	var lastErr error
	var triedModels []string
	var firstModelName string

	klog.Infof("=== 模型自动切换开始 ===")

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 1. 获取可用的模型池
		models, err := p.provider.GetModelPool(ctx, p.modelNames)
		if err != nil {
			klog.Errorf("ProxyChatModel: failed to get model pool: %v", err)
			return nil, fmt.Errorf("failed to get available models: %w", err)
		}

		// 列出所有可用模型
		availableModelNames := make([]string, len(models))
		for i, m := range models {
			availableModelNames[i] = m.APIKeyName
		}
		klog.Infof("当前可用模型池: %v", availableModelNames)

		// 过滤掉已经尝试过的模型
		var availableModels []*ModelWithMetadata
		for _, m := range models {
			tried := false
			for _, name := range triedModels {
				if m.APIKeyName == name {
					tried = true
					break
				}
			}
			if !tried {
				availableModels = append(availableModels, m)
			}
		}

		if len(availableModels) == 0 {
			klog.Errorf("ProxyChatModel: no available models left, all %d models failed", len(triedModels))
			return nil, fmt.Errorf("all models unavailable, last error: %w", lastErr)
		}

		// 2. 使用第一个可用模型
		model := availableModels[0]
		triedModels = append(triedModels, model.APIKeyName)

		// 记录首次使用的模型
		if attempt == 0 {
			firstModelName = model.APIKeyName
		}

		if attempt > 0 {
			// 明确的切换日志
			klog.Infof(">>> 模型切换: 从 [%s] 切换到 [%s] (第 %d/%d 次尝试)",
				firstModelName, model.APIKeyName, attempt+1, maxAttempts)
		} else {
			klog.Infof(">>> 使用模型: [%s] (第 %d/%d 次尝试)",
				model.APIKeyName, attempt+1, maxAttempts)
		}

		// 3. 绑定工具
		p.toolBinder.BindToModel(&model.ChatModel)

		// 4. 执行请求
		result, err := executor(model)
		if err == nil {
			// 成功，记录用量和请求
			if msg, ok := result.(*schema.Message); ok && msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
				p.recordUsage(ctx, model.LLMModel, msg.ResponseMeta.Usage)
			}
			if model.APIKeyID > 0 {
				if recordErr := p.provider.apiKeyService.RecordRequest(ctx, model.APIKeyID, true); recordErr != nil {
					klog.Warningf("ProxyChatModel: failed to record request for APIKeyID %d: %v", model.APIKeyID, recordErr)
				}
			}

			if attempt == 0 {
				klog.Infof("=== 模型执行成功 [%s] (无需切换) ===", model.APIKeyName)
			} else {
				klog.Infof("=== 模型执行成功 [%s] (经过 %d 次切换) ===", model.APIKeyName, attempt)
			}
			return result, nil
		}

		lastErr = err

		// 判断错误类型
		errorType := "未知错误"
		if p.rateLimiter.IsRateLimitError(err) {
			errorType = "Rate Limit"
		} else if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			errorType = "超时"
		} else if strings.Contains(strings.ToLower(err.Error()), "connection") {
			errorType = "连接错误"
		}

		klog.Warningf(">>> 模型 [%s] 失败: %v (错误类型: %s)", model.APIKeyName, err, errorType)

		// 5. 检查是否为可重试的错误
		if !p.isRetryableError(err) {
			klog.Errorf("=== 非可重试错误，终止切换: %v ===", err)
			return nil, err
		}

		// 6. 标记失败模型为不可用（如果是 rate limit 错误）
		if p.rateLimiter.IsRateLimitError(err) {
			if markErr := p.provider.MarkModelUnavailable(ctx, model.APIKeyName, time.Now().Add(2*time.Minute)); markErr != nil {
				klog.Warningf("ProxyChatModel: failed to mark model %s unavailable: %v", model.APIKeyName, markErr)
			}
			klog.Infof(">>> 模型 [%s] 已标记为不可用（Rate Limit）", model.APIKeyName)
		}

		// 7. 记录失败请求
		if model.APIKeyID > 0 {
			if recordErr := p.provider.apiKeyService.RecordRequest(ctx, model.APIKeyID, false); recordErr != nil {
				klog.Warningf("ProxyChatModel: failed to record failed request for APIKeyID %d: %v", model.APIKeyID, recordErr)
			}
		}

		// 8. 继续下一次尝试
		if attempt+1 < maxAttempts {
			klog.Infof("--- 准备尝试下一个模型 ---")
		}
	}

	// 所有尝试都失败
	klog.Errorf("=== 模型自动切换失败，所有 %d 个模型均不可用 ===", maxAttempts)
	return nil, fmt.Errorf("all %d model attempts failed, last error: %w", maxAttempts, lastErr)
}

// isRetryableError 判断错误是否可重试
func (p *ProxyChatModel) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Rate Limit 错误是可重试的
	if p.rateLimiter.IsRateLimitError(err) {
		return true
	}

	errStr := strings.ToLower(err.Error())

	// 网络相关错误
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"timeout",
		"network",
		"dns",
		"tcp",
		"tls",
		"ssl",
	}

	for _, keyword := range networkErrors {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
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

// BindTools 实现 model.ChatModel 接口
func (p *ProxyChatModel) BindTools(tools []*schema.ToolInfo) error {
	return p.toolBinder.BindTools(tools)
}

// WithTools 适配 model.ToolCallingChatModel 接口
func (p *ProxyChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	p.BindTools(tools)
	return p, nil
}
