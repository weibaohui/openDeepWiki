package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/opendeepwiki/backend/internal/pkg/agents"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// ============ Agent LLM 执行能力 ============

// ConversationOptions 会话执行配置
type ConversationOptions struct {
	// ConversationHistory 对话历史（多轮对话场景）
	ConversationHistory []llm.ChatMessage `json:"conversation_history,omitempty"`

	// MaxSteps 最大执行步骤数（覆盖 Agent.RuntimePolicy.MaxSteps）
	// 0 表示使用 Agent 配置的默认值
	MaxSteps int `json:"max_steps,omitempty"`

	// Temperature LLM 温度参数（0.0-1.0）
	// 0 表示使用默认值
	Temperature float64 `json:"temperature,omitempty"`

	// BasePath 基础路径（用于工具执行，如文件操作的根目录）
	BasePath string `json:"base_path,omitempty"`
}

// ConversationResult 会话执行结果
type ConversationResult struct {
	// Content 最终响应内容（LLM 的 assistant message content）
	Content string `json:"content"`

	// Messages 完整对话历史（可用于多轮对话）
	Messages []llm.ChatMessage `json:"messages"`

	// Steps 实际执行步骤数
	Steps int `json:"steps"`

	// ToolCalls 调用的工具摘要
	ToolCalls []ToolCallSummary `json:"tool_calls"`

	// Usage Token 使用统计
	Usage *LLMUsage `json:"usage,omitempty"`

	// AgentName 使用的 Agent 名称
	AgentName string `json:"agent_name"`

	// StartTime 执行开始时间
	StartTime time.Time `json:"start_time"`

	// EndTime 执行结束时间
	EndTime time.Time `json:"end_time"`
}

// ToolCallSummary 工具调用摘要
type ToolCallSummary struct {
	ToolName string `json:"tool_name"`
	Count    int    `json:"count"`
}

// LLMUsage LLM Token 使用统计
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// executionContext 执行上下文（内部使用）
type executionContext struct {
	agent          *agents.Agent
	messages       []llm.ChatMessage
	availableTools []llm.Tool
	toolCallCounts map[string]int
	step           int
	maxSteps       int
	basePath       string
	totalUsage     *LLMUsage
}

// ExecuteConversation 执行 Agent 会话
//
// 参数：
//   - ctx: 上下文（用于取消、超时等）
//   - agentName: Agent 名称
//   - userMessage: 用户消息
//   - options: 会话配置（可选）
//
// 返回：
//   - *ConversationResult: 会话结果
//   - error: 错误信息
func (e *Executor) ExecuteConversation(
	ctx context.Context,
	agentName string,
	userMessage string,
	options *ConversationOptions,
) (*ConversationResult, error) {
	klog.V(6).Infof("Agent conversation started: agent=%s, message=%s", agentName, userMessage)
	klog.V(6).Infof("Agent conversation options: %s", utils.ToJSON(options))

	// 1. 获取 Agent
	agent, err := e.manager.Registry.Get(agentName)
	if err != nil {
		klog.Errorf("Agent not found: %s", agentName)
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	// 2. 设置默认 options
	if options == nil {
		options = &ConversationOptions{}
	}

	// 3. 构造执行上下文
	execCtx := &executionContext{
		agent:          agent,
		messages:       e.buildMessages(agent, userMessage, options.ConversationHistory),
		availableTools: e.filterTools(&agent.SkillPolicy),
		toolCallCounts: make(map[string]int),
		basePath:       options.BasePath,
		totalUsage:     &LLMUsage{},
	}

	klog.V(6).Infof("execCtx.basePath=%s", execCtx.basePath)

	// 4. 确定 MaxSteps
	execCtx.maxSteps = agent.RuntimePolicy.MaxSteps
	if options.MaxSteps > 0 {
		execCtx.maxSteps = options.MaxSteps
	}

	klog.V(6).Infof("Agent max steps: %d", execCtx.maxSteps)
	klog.V(6).Infof("Available tools: %d", len(execCtx.availableTools))

	// 5. 执行多轮对话循环
	startTime := time.Now()
	var assistantMessage llm.ChatMessage

	for execCtx.step < execCtx.maxSteps {
		klog.V(6).Infof("Executing step %d/%d, agent=%s",
			execCtx.step+1, execCtx.maxSteps, agentName)

		// 5.1 调用 LLM
		response, err := e.llmClient.ChatWithTools(ctx, execCtx.messages, execCtx.availableTools)
		if err != nil {
			klog.Errorf("LLM call failed at step %d: %v", execCtx.step+1, err)
			return nil, fmt.Errorf("LLM call failed at step %d: %w", execCtx.step+1, err)
		}

		// 5.2 记录 Usage
		execCtx.trackUsage(response)

		if len(response.Choices) == 0 {
			klog.Errorf("LLM response choices is empty at step %d", execCtx.step+1)
			return nil, fmt.Errorf("LLM response choices is empty at step %d", execCtx.step+1)
		}
		klog.V(6).Infof("Step %d: LLM response: %s", execCtx.step+1, utils.ToJSON(response))
		// 5.3 提取 Assistant message
		assistantMessage = llm.ChatMessage{
			Role:      "assistant",
			Content:   response.Choices[0].Message.Content,
			ToolCalls: response.Choices[0].Message.ToolCalls,
		}
		execCtx.messages = append(execCtx.messages, assistantMessage)

		klog.V(6).Infof("Step %d: LLM returned %d tool calls",
			execCtx.step+1, len(assistantMessage.ToolCalls))

		// 5.4 检查是否有 Tool Calls
		if len(assistantMessage.ToolCalls) == 0 {
			// 没有 tool calls，退出循环
			klog.V(6).Infof("Step %d: No tool calls, ending conversation", execCtx.step+1)
			break
		}

		// 5.5 执行 Tool Calls
		toolResults := e.executeToolCalls(ctx, assistantMessage.ToolCalls, execCtx.basePath)
		klog.V(6).Infof("Step %d: Tool 执行：%s\n 执行结果: %s", execCtx.step+1, utils.ToJSON(assistantMessage.ToolCalls), utils.ToJSON(toolResults))
		// 5.6 将 Tool Results 转换为 Messages
		for i, result := range toolResults {
			execCtx.messages = append(execCtx.messages, llm.ChatMessage{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: assistantMessage.ToolCalls[i].ID,
			})

			// 记录工具调用
			toolName := assistantMessage.ToolCalls[i].Function.Name
			execCtx.trackToolCall(toolName)

			klog.V(6).Infof("Tool call: %s, error: %v, result length: %d",
				toolName, result.IsError, len(result.Content))
		}

		execCtx.step++
	}

	endTime := time.Now()

	// 6. 构造 Tool Call Summary
	var toolCallSummaries []ToolCallSummary
	for toolName, count := range execCtx.toolCallCounts {
		toolCallSummaries = append(toolCallSummaries, ToolCallSummary{
			ToolName: toolName,
			Count:    count,
		})
	}

	klog.V(6).Infof("Agent conversation completed: agent=%s, steps=%d, tokens=%d",
		agentName, execCtx.step, execCtx.totalUsage.TotalTokens)

	// 7. 返回结果
	return &ConversationResult{
		Content:   assistantMessage.Content,
		Messages:  execCtx.messages,
		Steps:     execCtx.step,
		ToolCalls: toolCallSummaries,
		Usage:     execCtx.totalUsage,
		AgentName: agentName,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

// filterTools 根据 Agent.SkillPolicy 过滤可用工具
func (e *Executor) filterTools(skillPolicy *agents.SkillPolicy) []llm.Tool {
	// 获取缓存的默认 tools
	allTools := e.defaultTools
	klog.V(6).Infof("Total default tools: %d", len(allTools))
	klog.V(6).Infof("allTools : %s", utils.ToJSON(allTools))

	klog.V(6).Infof("Skill policy: %s", utils.ToJSON(skillPolicy))

	// 如果 Allow 列表为空且 Deny 列表为空，返回所有 tools
	if len(skillPolicy.Allow) == 0 && len(skillPolicy.Deny) == 0 {
		return allTools
	}

	// 构建 allow map（快速查找）
	allowMap := make(map[string]bool)
	for _, toolName := range skillPolicy.Allow {
		allowMap[toolName] = true
	}

	// 构建 deny map
	denyMap := make(map[string]bool)
	for _, toolName := range skillPolicy.Deny {
		denyMap[toolName] = true
	}

	// 过滤 tools
	var filteredTools []llm.Tool
	for _, tool := range allTools {
		toolName := tool.Function.Name

		// 如果在 deny 列表中，跳过（deny 优先级最高）
		if denyMap[toolName] {
			continue
		}

		// 如果 allow 列表非空且不在 allow 列表中，跳过
		if len(skillPolicy.Allow) > 0 && !allowMap[toolName] {
			continue
		}

		filteredTools = append(filteredTools, tool)
	}

	klog.V(6).Infof("Filtered tools: %d -> %d", len(allTools), len(filteredTools))
	return filteredTools
}

// buildMessages 构造初始消息
func (e *Executor) buildMessages(
	agent *agents.Agent,
	userMessage string,
	history []llm.ChatMessage,
) []llm.ChatMessage {
	messages := []llm.ChatMessage{}

	// 1. System message
	messages = append(messages, llm.ChatMessage{
		Role:    "system",
		Content: agent.SystemPrompt,
	})

	// 2. History（多轮对话）
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// 3. User message
	messages = append(messages, llm.ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// executeToolCalls 执行工具调用
func (e *Executor) executeToolCalls(
	ctx context.Context,
	toolCalls []llm.ToolCall,
	basePath string,
) []llm.ToolResult {
	return e.toolExecutor.ExecuteAll(ctx, toolCalls, basePath)
}

// trackToolCall 记录工具调用
func (ctx *executionContext) trackToolCall(toolName string) {
	ctx.toolCallCounts[toolName]++
}

// trackUsage 记录 Token 使用
func (ctx *executionContext) trackUsage(response *llm.ChatResponse) {
	if ctx.totalUsage != nil {
		ctx.totalUsage.PromptTokens += response.Usage.PromptTokens
		ctx.totalUsage.CompletionTokens += response.Usage.CompletionTokens
		ctx.totalUsage.TotalTokens += response.Usage.TotalTokens
	}
}
