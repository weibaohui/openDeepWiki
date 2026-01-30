package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// Executor Skill 执行器
type Executor struct {
	registry Registry
}

// NewExecutor 创建 Skill 执行器
func NewExecutor(registry Registry) *Executor {
	return &Executor{registry: registry}
}

// Execute 执行 Tool Call
func (e *Executor) Execute(ctx context.Context, toolCall llm.ToolCall) (llm.ToolResult, error) {
	// 从 Registry 获取 Skill
	skill, err := e.registry.Get(toolCall.Function.Name)
	if err != nil {
		return llm.ToolResult{
			Content: fmt.Sprintf("Skill not found: %s", toolCall.Function.Name),
			IsError: true,
		}, nil
	}

	// 检查 Skill 是否已启用
	if !e.registry.IsEnabled(toolCall.Function.Name) {
		return llm.ToolResult{
			Content: fmt.Sprintf("Skill is disabled: %s", toolCall.Function.Name),
			IsError: true,
		}, nil
	}

	// 执行 Skill
	result, err := skill.Execute(ctx, json.RawMessage(toolCall.Function.Arguments))
	if err != nil {
		return llm.ToolResult{
			Content: fmt.Sprintf("Execution failed: %v", err),
			IsError: true,
		}, nil
	}

	// 序列化结果
	content, err := json.Marshal(result)
	if err != nil {
		return llm.ToolResult{
			Content: fmt.Sprintf("Failed to serialize result: %v", err),
			IsError: true,
		}, nil
	}

	return llm.ToolResult{
		Content: string(content),
		IsError: false,
	}, nil
}

// ExecuteMultiple 执行多个 Tool Calls
func (e *Executor) ExecuteMultiple(ctx context.Context, toolCalls []llm.ToolCall) []llm.ToolResult {
	results := make([]llm.ToolResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		result, err := e.Execute(ctx, tc)
		if err != nil {
			result = llm.ToolResult{
				Content: fmt.Sprintf("Execution error: %v", err),
				IsError: true,
			}
		}
		results = append(results, result)
	}

	return results
}

// Ensure Executor 实现了 llm.ToolExecutor 接口
var _ llm.ToolExecutor = (*Executor)(nil)
