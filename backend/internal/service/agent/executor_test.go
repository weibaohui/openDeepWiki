package agent

import (
	"testing"

	"github.com/opendeepwiki/backend/internal/pkg/agents"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/stretchr/testify/assert"
)

// 创建测试用的 Executor
func createTestExecutor() *Executor {
	return &Executor{
		defaultTools: llm.DefaultTools(),
	}
}

// 测试 filterTools - 只允许特定工具
func TestFilterTools_AllowOnly(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{
		Allow: []string{"read_file", "search_files"},
	}

	filtered := e.filterTools(skillPolicy)

	assert.Equal(t, 2, len(filtered), "应只保留允许的工具")

	toolNames := extractToolNames(filtered)
	assert.Contains(t, toolNames, "read_file", "应包含 read_file")
	assert.Contains(t, toolNames, "search_files", "应包含 search_files")
	assert.NotContains(t, toolNames, "execute_bash", "不应包含 execute_bash")
}

// 测试 filterTools - 拒绝特定工具
func TestFilterTools_DenyOnly(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{
		Deny: []string{"execute_bash", "count_lines"},
	}

	filtered := e.filterTools(skillPolicy)

	toolNames := extractToolNames(filtered)
	assert.NotContains(t, toolNames, "execute_bash", "不应包含 execute_bash")
	assert.NotContains(t, toolNames, "count_lines", "不应包含 count_lines")
	assert.Contains(t, toolNames, "read_file", "应包含 read_file")
}

// 测试 filterTools - Deny 优先级高于 Allow
func TestFilterTools_DenyTakesPriority(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{
		Allow: []string{"read_file", "execute_bash"},
		Deny:  []string{"execute_bash"},
	}

	filtered := e.filterTools(skillPolicy)

	toolNames := extractToolNames(filtered)
	assert.Contains(t, toolNames, "read_file", "应包含 read_file")
	assert.NotContains(t, toolNames, "execute_bash", "Deny 优先级高于 Allow，不应包含 execute_bash")
}

// 测试 filterTools - 无过滤规则
func TestFilterTools_NoFilters(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{}

	filtered := e.filterTools(skillPolicy)

	allTools := llm.DefaultTools()
	assert.Equal(t, len(allTools), len(filtered), "无过滤规则时，应返回所有工具")
}

// 测试 filterTools - Allow 和 Deny 都为空
func TestFilterTools_EmptyAllowAndDeny(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{
		Allow: []string{},
		Deny:  []string{},
	}

	filtered := e.filterTools(skillPolicy)

	allTools := llm.DefaultTools()
	assert.Equal(t, len(allTools), len(filtered), "Allow 和 Deny 都为空时，应返回所有工具")
}

// 测试 buildMessages - 基本消息构造
func TestBuildMessages_Basic(t *testing.T) {
	e := createTestExecutor()

	agent := &agents.Agent{
		SystemPrompt: "You are a helpful assistant.",
	}

	messages := e.buildMessages(agent, "Hello, world!", nil)

	assert.Equal(t, 2, len(messages), "应包含 system 和 user 消息")
	assert.Equal(t, "system", messages[0].Role, "第一条应为 system 消息")
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content, "System prompt 应正确")
	assert.Equal(t, "user", messages[1].Role, "第二条应为 user 消息")
	assert.Equal(t, "Hello, world!", messages[1].Content, "User message 应正确")
}

// 测试 buildMessages - 带历史记录
func TestBuildMessages_WithHistory(t *testing.T) {
	e := createTestExecutor()

	agent := &agents.Agent{
		SystemPrompt: "You are a helpful assistant.",
	}

	history := []llm.ChatMessage{
		{Role: "user", Content: "Question 1"},
		{Role: "assistant", Content: "Answer 1"},
	}

	messages := e.buildMessages(agent, "Question 2", history)

	assert.Equal(t, 4, len(messages), "应包含 system、history（2条）和 user 消息")
	assert.Equal(t, "system", messages[0].Role)
	assert.Equal(t, "user", messages[1].Role)
	assert.Equal(t, "assistant", messages[2].Role)
	assert.Equal(t, "user", messages[3].Role)
	assert.Equal(t, "Question 2", messages[3].Content)
}

// 测试 extractToolNames（辅助函数）
func extractToolNames(tools []llm.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Function.Name
	}
	return names
}

// 测试 trackToolCall
func TestTrackToolCall(t *testing.T) {
	ctx := &executionContext{
		toolCallCounts: make(map[string]int),
	}

	ctx.trackToolCall("read_file")
	assert.Equal(t, 1, ctx.toolCallCounts["read_file"], "read_file 应计数 1")

	ctx.trackToolCall("read_file")
	assert.Equal(t, 2, ctx.toolCallCounts["read_file"], "read_file 应计数 2")

	ctx.trackToolCall("search_files")
	assert.Equal(t, 1, ctx.toolCallCounts["search_files"], "search_files 应计数 1")
}

// 测试 trackUsage
func TestTrackUsage(t *testing.T) {
	ctx := &executionContext{
		totalUsage: &LLMUsage{},
	}

	response1 := &llm.ChatResponse{
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	response2 := &llm.ChatResponse{
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
		},
	}

	ctx.trackUsage(response1)
	assert.Equal(t, 100, ctx.totalUsage.PromptTokens)
	assert.Equal(t, 50, ctx.totalUsage.CompletionTokens)
	assert.Equal(t, 150, ctx.totalUsage.TotalTokens)

	ctx.trackUsage(response2)
	assert.Equal(t, 300, ctx.totalUsage.PromptTokens)
	assert.Equal(t, 150, ctx.totalUsage.CompletionTokens)
	assert.Equal(t, 450, ctx.totalUsage.TotalTokens)
}

// 测试 ConversationOptions 默认值
func TestConversationOptions_DefaultValues(t *testing.T) {
	options := &ConversationOptions{}

	assert.Equal(t, 0, options.MaxSteps, "MaxSteps 默认值应为 0")
	assert.Equal(t, 0.0, options.Temperature, "Temperature 默认值应为 0.0")
	assert.Nil(t, options.ConversationHistory, "ConversationHistory 默认值应为 nil")
	assert.Equal(t, "", options.BasePath, "BasePath 默认值应为空字符串")
}

// 测试 ConversationResult 字段
func TestConversationResult_Fields(t *testing.T) {
	result := &ConversationResult{
		Content:   "Test response",
		Steps:     5,
		AgentName: "test-agent",
	}

	assert.Equal(t, "Test response", result.Content)
	assert.Equal(t, 5, result.Steps)
	assert.Equal(t, "test-agent", result.AgentName)
	assert.Nil(t, result.ToolCalls)
	assert.Nil(t, result.Usage)
}

// 测试 ToolCallSummary
func TestToolCallSummary(t *testing.T) {
	summary := ToolCallSummary{
		ToolName: "read_file",
		Count:    3,
	}

	assert.Equal(t, "read_file", summary.ToolName)
	assert.Equal(t, 3, summary.Count)
}

// 测试 LLMUsage
func TestLLMUsage(t *testing.T) {
	usage := LLMUsage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	assert.Equal(t, 1000, usage.PromptTokens)
	assert.Equal(t, 500, usage.CompletionTokens)
	assert.Equal(t, 1500, usage.TotalTokens)
}

// 测试 ExecuteConversation 的集成测试（需要 mock）
func TestExecuteConversation_Integration(t *testing.T) {
	// 注意：这个测试需要真实的 Agent Manager 和 LLM Client
	// 在实际环境中，可以使用 mock 或者跳过这个测试

	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// TODO: 实现 mock Agent Manager 和 LLM Client
	// 创建测试 Agent
	_ = &agents.Agent{
		Name:         "test-agent",
		SystemPrompt: "You are a helpful assistant.",
		RuntimePolicy: agents.RuntimePolicy{
			MaxSteps:  10,
			RiskLevel: "read",
		},
		SkillPolicy: agents.SkillPolicy{
			Allow: []string{"read_file", "search_files"},
		},
	}

	// 创建测试用的 manager（这里需要 mock）
	// manager := &mockAgentManager{agents: map[string]*agents.Agent{"test-agent": agent}}

	// e := &Executor{
	// 	manager:      manager,
	// 	llmClient:    &mockLLMClient{},
	// 	toolExecutor: llm.NewSafeExecutor(".", llm.DefaultExecutorConfig()),
	// 	defaultTools: llm.DefaultTools(),
	// }

	// result, err := e.ExecuteConversation(context.Background(), "test-agent", "Hello", nil)

	// require.NoError(t, err)
	// assert.Equal(t, "test-agent", result.AgentName)
	// assert.NotEmpty(t, result.Content)
}

// 测试工具过滤的性能
func TestFilterTools_Performance(t *testing.T) {
	e := createTestExecutor()

	// 包含大量 Allow 和 Deny 的场景
	skillPolicy := &agents.SkillPolicy{
		Allow: []string{
			"read_file", "search_files", "search_text", "list_dir", "file_stat",
			"git_log", "git_status", "git_diff", "extract_functions", "get_file_tree",
		},
		Deny: []string{"execute_bash"},
	}

	// 多次调用，确保性能稳定
	for i := 0; i < 100; i++ {
		filtered := e.filterTools(skillPolicy)
		assert.Equal(t, 10, len(filtered))
	}
}

// 测试工具名称不存在的场景
func TestFilterTools_NonexistentTools(t *testing.T) {
	e := createTestExecutor()

	skillPolicy := &agents.SkillPolicy{
		Allow: []string{"nonexistent_tool_1", "nonexistent_tool_2"},
	}

	filtered := e.filterTools(skillPolicy)

	// 如果 Allow 中的工具不存在，应返回空列表
	assert.Equal(t, 0, len(filtered), "不存在工具时，应返回空列表")
}
