package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/pkg/skills"
)

// Provider 内置 Provider
type Provider struct {
	creators map[string]SkillCreator
}

// SkillCreator Skill 创建函数
type SkillCreator func(config skills.SkillConfig) (skills.Skill, error)

// NewProvider 创建 Builtin Provider
func NewProvider() *Provider {
	return &Provider{
		creators: make(map[string]SkillCreator),
	}
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
	return "builtin"
}

// Register 注册 Skill 创建器
func (p *Provider) Register(name string, creator SkillCreator) {
	p.creators[name] = creator
}

// Create 创建 Skill
func (p *Provider) Create(config skills.SkillConfig) (skills.Skill, error) {
	// Builtin Provider 直接使用配置中的 name 查找创建器
	creator, exists := p.creators[config.Name]
	if !exists {
		// 如果没有预注册的创建器，尝试使用配置创建一个通用 BuiltinSkill
		// 这需要配合 ExecuteFunc 使用
		return nil, fmt.Errorf("builtin skill %q not found, please register it first", config.Name)
	}

	return creator(config)
}

// BuiltinSkill 内置 Skill 实现
type BuiltinSkill struct {
	name        string
	description string
	parameters  llm.ParameterSchema
	fn          ExecuteFunc
	providerType string
}

// ExecuteFunc 执行函数类型
type ExecuteFunc func(ctx context.Context, args json.RawMessage) (interface{}, error)

// NewBuiltinSkill 创建内置 Skill
func NewBuiltinSkill(name, description string, parameters llm.ParameterSchema, fn ExecuteFunc) *BuiltinSkill {
	return &BuiltinSkill{
		name:        name,
		description: description,
		parameters:  parameters,
		fn:          fn,
		providerType: "builtin",
	}
}

// Name 返回名称
func (s *BuiltinSkill) Name() string {
	return s.name
}

// Description 返回描述
func (s *BuiltinSkill) Description() string {
	return s.description
}

// Parameters 返回参数定义
func (s *BuiltinSkill) Parameters() skills.ParameterSchema {
	return s.parameters
}

// Execute 执行
func (s *BuiltinSkill) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	if s.fn == nil {
		return nil, fmt.Errorf("execute function not implemented")
	}
	return s.fn(ctx, args)
}

// ProviderType 返回 Provider 类型
func (s *BuiltinSkill) ProviderType() string {
	return s.providerType
}
