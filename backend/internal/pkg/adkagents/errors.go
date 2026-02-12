// Package adkagents 提供 ADK Agent 的 YAML 配置化管理
package adkagents

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

// 错误定义
var (
	// ErrAgentNotFound Agent 不存在
	ErrAgentNotFound = errors.New("agent not found")

	// ErrInvalidConfig 配置文件无效
	ErrInvalidConfig = errors.New("invalid agent config")

	// ErrInvalidName Agent 名称格式无效
	ErrInvalidName = errors.New("invalid agent name")

	// ErrToolNotFound 工具不存在
	ErrToolNotFound = errors.New("tool not found")

	// ErrModelNotFound 模型不存在
	ErrModelNotFound = errors.New("model not found")

	// ErrAgentDirNotFound Agent 目录不存在
	ErrAgentDirNotFound = errors.New("agents directory not found")

	// ErrAgentAlreadyExists Agent 已存在
	ErrAgentAlreadyExists = errors.New("agent already exists")

	// ErrConfigNotFound 配置文件不存在
	ErrConfigNotFound = errors.New("config file not found")

	// ErrAPIKeyNotFound API Key 不存在
	ErrAPIKeyNotFound = errors.New("api key not found")

	// ErrModelUnavailable 模型不可用
	ErrModelUnavailable = errors.New("model unavailable")

	// ErrNoAvailableModel 没有可用模型
	ErrNoAvailableModel = errors.New("no available model")

	// ErrRateLimitExceeded 速率限制超出
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// FallbackToDefault 兜底到默认模型的错误类型
type FallbackToDefault struct {
	Model *openai.ChatModel
}

func (e *FallbackToDefault) Error() string {
	return "fallback to default model"
}

// ModelUnavailableDetail 模型不可用详细信息
type ModelUnavailableDetail struct {
	ModelName string
	ResetAt   time.Time
	Reason    string
}

func (e *ModelUnavailableDetail) Error() string {
	return fmt.Sprintf("model %s unavailable: %s, reset at %v", e.ModelName, e.Reason, e.ResetAt)
}
