package adkagents

import (
	"context"
	"time"

	"k8s.io/klog/v2"
)

// ModelSwitcher 模型切换器
type ModelSwitcher struct {
	apiKeyService APIKeyService
	rateLimiter   *RateLimiter
}

// NewModelSwitcher 创建模型切换器
func NewModelSwitcher(apiKeyService APIKeyService) *ModelSwitcher {
	return &ModelSwitcher{
		apiKeyService: apiKeyService,
	}
}

// SetRateLimiter 设置速率限制器
func (s *ModelSwitcher) SetRateLimiter(rateLimiter *RateLimiter) {
	s.rateLimiter = rateLimiter
}

// CallWithRetry 使用模型切换重试机制调用
func (s *ModelSwitcher) CallWithRetry(
	ctx context.Context,
	provider ModelProvider,
	poolNames []string,
	fn func(*ModelWithMetadata) (interface{}, error),
) (interface{}, error) {
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		klog.V(6).Infof("ModelSwitcher.CallWithRetry: attempt %d/%d", attempt+1, maxRetries)

		// 获取模型池
		models, err := provider.GetModelPool(ctx, poolNames)
		if err != nil {
			klog.Errorf("ModelSwitcher.CallWithRetry: failed to get model pool: %v", err)
			return nil, err
		}

		if len(models) == 0 {
			klog.Error("ModelSwitcher.CallWithRetry: no available models")
			return nil, ErrNoAvailableModel
		}

		// 使用第一个可用模型
		currentModel := models[0]
		klog.V(6).Infof("ModelSwitcher.CallWithRetry: using model %s", currentModel.APIKeyName)

		// 调用函数
		result, err := fn(currentModel)
		if err != nil {
			klog.V(6).Infof("ModelSwitcher.CallWithRetry: error occurred: %v", err)

			// 检查是否为 Rate Limit 错误
			if s.rateLimiter != nil && s.rateLimiter.IsRateLimitError(err) {
				klog.Warningf("ModelSwitcher.CallWithRetry: rate limit hit for model %s", currentModel.APIKeyName)

				// 处理 Rate Limit
				_ = s.rateLimiter.HandleRateLimit(ctx, currentModel.APIKeyName, err, time.Hour)

				// 如果还有重试机会，继续下一次尝试
				if attempt+1 < maxRetries {
					klog.Infof("ModelSwitcher.CallWithRetry: retrying with next model...")
					time.Sleep(1 * time.Second) // 等待 1 秒后重试
					continue
				}

				return nil, ErrAllModelsUnavailable
			}

			// 非 Rate Limit 错误，直接返回
			return nil, err
		}

		// 成功，记录使用情况
		if currentModel.APIKeyID > 0 {
			s.apiKeyService.RecordRequest(ctx, currentModel.APIKeyID, true)
		}

		return result, nil
	}

	return nil, ErrMaxRetriesExceeded
}

// ErrAllModelsUnavailable 所有模型都不可用
var ErrAllModelsUnavailable = ErrRateLimitExceeded

// ErrMaxRetriesExceeded 超过最大重试次数
var ErrMaxRetriesExceeded = ErrRateLimitExceeded
