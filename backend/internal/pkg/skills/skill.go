package skills

import (
	"context"
	"encoding/json"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// Skill 技能接口，所有技能必须实现
type Skill interface {
	// Name 返回技能唯一名称
	// 约束：全局唯一，符合 [a-zA-Z0-9_-]+ 格式
	Name() string

	// Description 返回技能描述
	// 供 LLM 理解该技能的用途
	Description() string

	// Parameters 返回参数 JSON Schema
	// 符合 JSON Schema Draft 7 规范
	Parameters() llm.ParameterSchema

	// Execute 执行技能
	// ctx: 上下文，包含超时控制
	// args: JSON 格式的参数，需根据 Parameters 解析
	// 返回: 执行结果（必须可 JSON 序列化）和错误
	Execute(ctx context.Context, args json.RawMessage) (interface{}, error)

	// ProviderType 返回提供者类型
	// 用于调试和监控
	ProviderType() string
}

// ParameterSchema 参数 JSON Schema 定义（别名，用于向后兼容）
type ParameterSchema = llm.ParameterSchema

// Property 单个参数属性（别名，用于向后兼容）
type Property = llm.Property
