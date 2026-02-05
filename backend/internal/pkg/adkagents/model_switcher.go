package adkagents

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"k8s.io/klog/v2"
)

// ModelSwitcher 模型切换器
type ModelSwitcher struct {
	apiKeyService APIKeyService
	modelProvider ModelProvider
}

// NewModelSwitcher 创建模型切换器
func NewModelSwitcher(apiKeyService APIKeyService) *ModelSwitcher {
	return &ModelSwitcher{
		apiKeyService: apiKeyService,
	}
}

// SetModelProvider 设置模型提供者
func (s *ModelSwitcher) SetModelProvider(provider ModelProvider) {
	s.modelProvider = provider
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
			if provider.IsRateLimitError(err) {
				klog.Warningf("ModelSwitcher.CallWithRetry: rate limit hit for model %s", currentModel.APIKeyName)

				// 解析重置时间
				resetTime := s.parseResetTime(err)
				if resetTime.IsZero() {
					// 如果没有明确的重置时间，设置默认为 1 小时后
					resetTime = time.Now().Add(time.Hour)
				}

				// 标记当前模型为不可用
				if err := provider.MarkModelUnavailable(currentModel.APIKeyName, resetTime); err != nil {
					klog.Errorf("ModelSwitcher.CallWithRetry: failed to mark model unavailable: %v", err)
				}

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

// parseResetTime 从错误中解析重置时间
func (s *ModelSwitcher) parseResetTime(err error) time.Time {
	if err == nil {
		return time.Time{}
	}

	errMsg := err.Error()

	// 尝试从错误消息中解析重置时间
	// 格式可能为：Try again in 60s, Retry after 1m, Reset at 2026-02-04 12:00:00
	patterns := []struct {
		pattern   string
		duration time.Duration
	}{
		{`Try again in (\d+)s`, time.Second},
		{`Retry after (\d+)s`, time.Second},
		{`Try again in (\d+)m`, time.Minute},
		{`Retry after (\d+)m`, time.Minute},
		{`Try again in (\d+)h`, time.Hour},
		{`Retry after (\d+)h`, time.Hour},
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) >= 2 {
			var duration int
			if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
				return time.Now().Add(time.Duration(duration) * p.duration)
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

	// 无法解析，返回零值
	return time.Time{}
}

// ErrAllModelsUnavailable 所有模型都不可用
var ErrAllModelsUnavailable = fmt.Errorf("all models unavailable")

// ErrMaxRetriesExceeded 超过最大重试次数
var ErrMaxRetriesExceeded = fmt.Errorf("max retries exceeded")
