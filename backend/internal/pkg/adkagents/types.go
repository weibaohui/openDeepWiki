package adkagents

import (
	"time"

	"github.com/cloudwego/eino/components/model"
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

// ModelProvider 模型提供者接口
type ModelProvider interface {
	// GetModel 获取指定名称的模型，name 为空时返回默认模型
	GetModel(name string) (model.ToolCallingChatModel, error)
	// DefaultModel 获取默认模型
	DefaultModel() model.ToolCallingChatModel
}

// Now 返回当前时间（用于测试）
var Now = func() time.Time {
	return time.Now()
}
