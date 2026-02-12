package adkagents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// RateLimiter 速率限制处理器
type RateLimiter struct {
	provider Provider
}

// Provider 模型提供者接口（避免与 types.go 中的 ModelProvider 冲突）
type Provider interface {
	MarkModelUnavailable(ctx context.Context, modelName string, resetTime time.Time) error
}

// NewRateLimiter 创建速率限制处理器
func NewRateLimiter(provider Provider) *RateLimiter {
	return &RateLimiter{
		provider: provider,
	}
}

// IsRateLimitError 判断错误是否为 Rate Limit 错误
func (r *RateLimiter) IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// 检查 HTTP 状态码
	if strings.Contains(errMsg, "429") {
		return true
	}

	// 检查错误消息关键词
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

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// ParseResetTime 从错误中解析重置时间
func (r *RateLimiter) ParseResetTime(err error) time.Time {
	if err == nil {
		return time.Time{}
	}

	errMsg := err.Error()

	// 解析持续时间模式：Try again in 60s, Retry after 1m, Try again in 2h
	durationPatterns := []struct {
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

	for _, pt := range durationPatterns {
		re := regexp.MustCompile(pt.pattern)
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) >= 2 {
			var duration int
			if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
				return time.Now().Add(time.Duration(duration) * pt.duration)
			}
		}
	}

	// 解析具体时间模式：Reset at 2026-02-04 12:00:00
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

// HandleRateLimit 处理 Rate Limit 错误
func (r *RateLimiter) HandleRateLimit(ctx context.Context, modelName string, err error, defaultResetDuration time.Duration) error {
	klog.Warningf("RateLimiter: rate limit hit for model %s: %v", modelName, err)

	resetTime := r.ParseResetTime(err)
	if resetTime.IsZero() {
		resetTime = time.Now().Add(defaultResetDuration)
	}

	if markErr := r.provider.MarkModelUnavailable(ctx, modelName, resetTime); markErr != nil {
		klog.Errorf("RateLimiter: failed to mark model unavailable: %v", markErr)
	}

	return err
}
