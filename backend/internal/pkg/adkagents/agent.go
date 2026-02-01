package adkagents

import (
	"time"
)

// AgentDefinition ADK Agent 定义（从 YAML 加载）
type AgentDefinition struct {
	// 元数据
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`

	// LLM 配置
	Model string `yaml:"model" json:"model"` // 模型名称或别名，空则使用默认

	// Agent 行为配置
	Instruction   string   `yaml:"instruction" json:"instruction"`      // System Prompt
	Tools         []string `yaml:"tools" json:"tools"`                  // 工具名称列表
	MaxIterations int      `yaml:"maxIterations" json:"max_iterations"` // 最大迭代次数

	// 可选配置
	Exit ExitConfig `yaml:"exit,omitempty" json:"exit,omitempty"` // 退出条件

	// 路径信息（运行时填充）
	Path     string    `json:"path"`       // 配置文件路径
	LoadedAt time.Time `json:"loaded_at"`  // 加载时间
}

// HasTool 检查 Agent 是否配置了指定工具
func (a *AgentDefinition) HasTool(toolName string) bool {
	for _, t := range a.Tools {
		if t == toolName {
			return true
		}
	}
	return false
}

// ToolCount 返回配置的工具数量
func (a *AgentDefinition) ToolCount() int {
	return len(a.Tools)
}
