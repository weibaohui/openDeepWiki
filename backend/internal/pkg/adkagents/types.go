package adkagents

import (
	"context"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// ExitConfig 退出条件配置
type ExitConfig struct {
	Type string `yaml:"type" json:"type"` // 退出类型，如 "tool_call"
}

// LoadResult 加载结果
type LoadResult struct {
	Agent  *AgentDefinition
	Error  error
	Action string // "created", "updated", "deleted", "failed"
}

// FileEvent 文件事件
type FileEvent struct {
	Type string // "create", "modify", "delete"
	Path string
}

// APIKeyService API Key 服务接口（避免循环导入）
type APIKeyService interface {
	MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error
	RecordRequest(ctx context.Context, apiKeyID uint, success bool) error
}

// TaskUsageService 任务用量记录接口（避免循环导入）
type TaskUsageService interface {
	RecordUsage(ctx context.Context, taskID uint, apiKeyName string, usage *schema.TokenUsage) error
}

// ModelWithMetadata 带有元数据的模型包装器
type ModelWithMetadata struct {
	openai.ChatModel
	APIKeyName string
	APIKeyID   uint
	LLMModel   string
}

// Name 返回模型名称
func (m *ModelWithMetadata) Name() string {
	return m.APIKeyName
}

// ModelProvider 模型提供者接口
type ModelProvider interface {
	// GetModel 获取指定名称的模型，name 为空时返回默认模型
	GetModel(name string) (*openai.ChatModel, error)
	// DefaultModel 获取默认模型
	DefaultModel() *openai.ChatModel
	// GetModelPool 获取模型池
	GetModelPool(ctx context.Context, names []string) ([]*ModelWithMetadata, error)
	// IsRateLimitError 判断是否为 Rate Limit 错误
	IsRateLimitError(err error) bool
	// MarkModelUnavailable 标记模型为不可用
	MarkModelUnavailable(modelName string, resetTime time.Time) error
	// GetNextModel 获取下一个可用模型
	GetNextModel(ctx context.Context, currentModelName string, poolNames []string) (*ModelWithMetadata, error)
}

// Now 返回当前时间（用于测试）
var Now = func() time.Time {
	return time.Now()
}
