package agents

import (
	"time"
)

// Agent Agent定义
type Agent struct {
	// 元数据
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description" json:"description"`

	// System Prompt
	SystemPrompt string `yaml:"systemPrompt" json:"system_prompt"`

	// MCP Policy
	McpPolicy McpPolicy `yaml:"mcp" json:"mcp_policy"`

	// Skill Policy
	SkillPolicy SkillPolicy `yaml:"skills" json:"skill_policy"`

	// Runtime Policy
	RuntimePolicy RuntimePolicy `yaml:"policies" json:"runtime_policy"`

	// 路径信息
	Path     string    `json:"path"` // 配置文件路径
	LoadedAt time.Time `json:"loaded_at"`
}

// McpPolicy MCP策略
type McpPolicy struct {
	Allowed  []string `yaml:"allowed" json:"allowed"`    // 允许的 MCP 列表
	MaxCalls int      `yaml:"maxCalls" json:"max_calls"` // 最大调用次数
}

// SkillPolicy Skill策略
type SkillPolicy struct {
	Allow []string `yaml:"allow" json:"allow"` // 显式允许的 Skills
	Deny  []string `yaml:"deny" json:"deny"`   // 显式禁止的 Skills
}

// RuntimePolicy 运行时策略
type RuntimePolicy struct {
	RiskLevel           string `yaml:"riskLevel" json:"risk_level"`                     // 风险等级：read / write / admin
	MaxSteps            int    `yaml:"maxSteps" json:"max_steps"`                       // 最大执行步骤数
	RequireConfirmation bool   `yaml:"requireConfirmation" json:"require_confirmation"` // 是否需要确认
}

// IsSkillAllowed 检查 Skill 是否被允许
func (a *Agent) IsSkillAllowed(skillName string) bool {
	// 如果在 deny 列表中，明确禁止
	for _, denied := range a.SkillPolicy.Deny {
		if denied == skillName {
			return false
		}
}

	// 如果 allow 列表为空，允许所有（除了 deny 的）
	if len(a.SkillPolicy.Allow) == 0 {
		return true
	}

	// 检查是否在 allow 列表中
	for _, allowed := range a.SkillPolicy.Allow {
		if allowed == skillName {
			return true
		}
	}

	return false
}

// IsMcpAllowed 检查 MCP 是否被允许
func (a *Agent) IsMcpAllowed(mcpName string) bool {
	// 如果 allowed 列表为空，允许所有
	if len(a.McpPolicy.Allowed) == 0 {
		return true
	}

	for _, allowed := range a.McpPolicy.Allowed {
		if allowed == mcpName {
			return true
		}
	}

	return false
}
