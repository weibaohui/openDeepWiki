package adkagents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// 预编译的正则表达式模式
var (
	durationPatterns = []struct {
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

	timePatterns = []string{
		`Reset at (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`,
		`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`,
	}
)

// precompiledRegexes 预编译的正则表达式（延迟初始化）
type precompiledRegexes struct {
	durationRegex     []*regexp.Regexp
	durationDurations []time.Duration
	timeRegex         []*regexp.Regexp
}

var compiledRegexes struct {
	once    sync.Once
	regexes *precompiledRegexes
}

// getCompiledRegexes 获取预编译的正则表达式
func getCompiledRegexes() *precompiledRegexes {
	compiledRegexes.once.Do(func() {
		durationRegex := make([]*regexp.Regexp, len(durationPatterns))
		durationDurations := make([]time.Duration, len(durationPatterns))

		for i, pt := range durationPatterns {
			durationRegex[i] = regexp.MustCompile(pt.pattern)
			durationDurations[i] = pt.duration
		}

		timeRegex := make([]*regexp.Regexp, len(timePatterns))
		for i, pattern := range timePatterns {
			timeRegex[i] = regexp.MustCompile(pattern)
		}

		compiledRegexes.regexes = &precompiledRegexes{
			durationRegex:     durationRegex,
			durationDurations: durationDurations,
			timeRegex:         timeRegex,
		}
	})
	return compiledRegexes.regexes
}

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

// ParseResetTime 从错误中解析重置时间（使用预编译的正则表达式）
func (r *RateLimiter) ParseResetTime(err error) time.Time {
	if err == nil {
		return time.Time{}
	}

	errMsg := err.Error()
	regexes := getCompiledRegexes()

	// 使用预编译的正则表达式解析持续时间
	for i, re := range regexes.durationRegex {
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) >= 2 {
			var duration int
			if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
				return time.Now().Add(time.Duration(duration) * regexes.durationDurations[i])
			}
		}
	}

	// 使用预编译的正则表达式解析具体时间
	for _, re := range regexes.timeRegex {
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
