# 009-Agent的LLM执行能力-设计.md

## 1. 概述

本文档描述如何为 Agent 系统实现 LLM 执行能力，使其能够带着 tools 发起 LLM 会话，并基于自身的 Policy 约束行为。

**设计目标**：
- 让 Agent 成为真正可用的智能体，能够执行 LLM 会话
- 复用现有的 LLM Client、ToolExecutor、Agent Manager
- 保持架构清晰、职责单一
- 提供良好的扩展性和可测试性

---

## 2. 核心设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Agent Executor                             │ │
│  │  - ExecuteConversation()                               │ │
│  │  - filterTools()                                       │ │
│  │  - executeToolCalls()                                  │ │
│  └──────────────────┬──────────────────────────────────────┘ │
│                     │                                         │
└─────────────────────┼───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
        ↓             ↓             ↓
┌──────────────┐ ┌─────────┐ ┌──────────────┐
│Agent Manager │ │LLM      │ │Tool Executor │
│(agents pkg)  │ │Client   │ │(llm pkg)     │
│              │ │         │ │              │
└──────────────┘ └─────────┘ └──────────────┘
```

**职责划分**：

- **Agent Manager**（`internal/pkg/agents/`）：负责 Agent 的注册、加载、查询，提供 Agent 定义
- **LLM Client**（`internal/pkg/llm/client.go`）：负责调用 LLM API
- **Tool Executor**（`internal/pkg/llm/executor.go`）：负责执行 tool calls
- **Agent Executor**（`internal/service/agent/executor.go`）：协调以上组件，实现 Agent LLM 执行能力

### 2.2 执行流程

```
用户请求
   │
   ├─1. 获取 Agent（通过 Agent Manager）
   │
   ├─2. 过滤 Tools（根据 Agent.SkillPolicy）
   │      ├─ 从 DefaultTools() 获取所有 tools
   │      ├─ 应用 Allow 策略
   │      └─ 应用 Deny 策略
   │
   ├─3. 构造 Messages
   │      ├─ System: Agent.SystemPrompt
   │      ├─ History: ConversationHistory（如果有）
   │      └─ User: UserMessage
   │
   ├─4. 执行多轮对话循环（直到 MaxSteps）
   │      │
   │      ├─4.1 调用 LLM
   │      │      ├─ 构造 ChatRequest
   │      │      └─ 调用 llm.Client.Chat()
   │      │
   │      ├─4.2 处理响应
   │      │      ├─ 提取 Assistant message
   │      │      └─ 添加到 Messages
   │      │
   │      ├─4.3 检查是否有 Tool Calls
   │      │      └─ 如果没有，退出循环
   │      │
   │      ├─4.4 执行 Tool Calls
   │      │      ├─ 调用 toolExecutor.ExecuteAll()
   │      │      └─ 获取 Tool Results
   │      │
   │      └─4.5 添加 Tool Results 到 Messages
   │
   └─5. 返回结果
          └─ 包含 Content、Messages、Steps、ToolCalls、Usage
```

---

## 3. 数据结构设计

### 3.1 会话配置（ConversationOptions）

```go
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
```

### 3.2 会话结果（ConversationResult）

```go
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
```

### 3.3 执行上下文（ExecutionContext）

```go
// executionContext 执行上下文（内部使用，不对外暴露）
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
```

---

## 4. 核心接口设计

### 4.1 Agent Executor 接口

```go
// Executor Agent 执行器
type Executor struct {
    cfg         *config.Config
    manager     *agents.Manager
    llmClient   *llm.Client
    toolExecutor *llm.SafeExecutor
    defaultTools []llm.Tool  // 缓存默认 tools
}
```

### 4.2 主方法：ExecuteConversation

```go
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
) (*ConversationResult, error)
```

### 4.3 辅助方法

```go
// filterTools 根据 Agent.SkillPolicy 过滤可用工具
func (e *Executor) filterTools(skillPolicy *agents.SkillPolicy) []llm.Tool

// executeToolCalls 执行工具调用
func (e *Executor) executeToolCalls(
    ctx context.Context,
    toolCalls []llm.ToolCall,
    basePath string,
) []llm.ToolResult

// buildMessages 构造初始消息
func (e *Executor) buildMessages(
    agent *agents.Agent,
    userMessage string,
    history []llm.ChatMessage,
) []llm.ChatMessage

// trackToolCall 记录工具调用
func (ctx *executionContext) trackToolCall(toolName string)

// trackUsage 记录 Token 使用
func (ctx *executionContext) trackUsage(usage llm.ChatResponse)
```

---

## 5. 实现细节

### 5.1 Tools 过滤逻辑

```go
func (e *Executor) filterTools(skillPolicy *agents.SkillPolicy) []llm.Tool {
    // 获取所有默认 tools
    allTools := llm.DefaultTools()

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

    return filteredTools
}
```

**过滤规则**：
1. Deny 优先级高于 Allow
2. 如果 Allow 非空，只保留 Allow 列表中的 tools
3. 如果 Allow 为空且 Deny 为空，返回所有 tools

### 5.2 会话执行循环

```go
func (e *Executor) ExecuteConversation(
    ctx context.Context,
    agentName string,
    userMessage string,
    options *ConversationOptions,
) (*ConversationResult, error) {
    // 1. 获取 Agent
    agent, err := e.manager.Get(agentName)
    if err != nil {
        return nil, fmt.Errorf("agent not found: %w", err)
    }

    // 2. 构造执行上下文
    execCtx := &executionContext{
        agent:          agent,
        messages:       e.buildMessages(agent, userMessage, options.ConversationHistory),
        availableTools: e.filterTools(&agent.SkillPolicy),
        toolCallCounts: make(map[string]int),
        basePath:       options.BasePath,
        totalUsage:     &LLMUsage{},
    }

    // 3. 确定 MaxSteps
    execCtx.maxSteps = agent.RuntimePolicy.MaxSteps
    if options.MaxSteps > 0 {
        execCtx.maxSteps = options.MaxSteps
    }

    // 4. 执行多轮对话循环
    startTime := time.Now()
    for execCtx.step < execCtx.maxSteps {
        klog.V(6).Infof("执行步骤 %d/%d, agent=%s",
            execCtx.step+1, execCtx.maxSteps, agentName)

        // 4.1 调用 LLM
        response, err := e.llmClient.Chat(ctx, llm.ChatRequest{
            Model:       e.cfg.LLM.Model,
            Messages:    execCtx.messages,
            Tools:       execCtx.availableTools,
            ToolChoice:  "auto",
            Temperature: options.Temperature,
        })
        if err != nil {
            return nil, fmt.Errorf("LLM call failed: %w", err)
        }

        // 4.2 记录 Usage
        execCtx.trackUsage(response)

        // 4.3 提取 Assistant message
        assistantMessage := llm.ChatMessage{
            Role:      "assistant",
            Content:   response.Choices[0].Message.Content,
            ToolCalls: response.Choices[0].Message.ToolCalls,
        }
        execCtx.messages = append(execCtx.messages, assistantMessage)

        // 4.4 检查是否有 Tool Calls
        if len(assistantMessage.ToolCalls) == 0 {
            // 没有 tool calls，退出循环
            break
        }

        // 4.5 执行 Tool Calls
        toolResults := e.executeToolCalls(ctx, assistantMessage.ToolCalls, execCtx.basePath)

        // 4.6 将 Tool Results 转换为 Messages
        for i, result := range toolResults {
            execCtx.messages = append(execCtx.messages, llm.ChatMessage{
                Role:       "tool",
                Content:    result.Content,
                ToolCallID: assistantMessage.ToolCalls[i].ID,
            })

            // 记录工具调用
            toolName := assistantMessage.ToolCalls[i].Function.Name
            execCtx.trackToolCall(toolName)

            klog.V(6).Infof("Tool call: %s, result: %s",
                toolName, result.Content)
        }

        execCtx.step++
    }

    endTime := time.Now()

    // 5. 构造 Tool Call Summary
    var toolCallSummaries []ToolCallSummary
    for toolName, count := range execCtx.toolCallCounts {
        toolCallSummaries = append(toolCallSummaries, ToolCallSummary{
            ToolName: toolName,
            Count:    count,
        })
    }

    // 6. 返回结果
    return &ConversationResult{
        Content:    assistantMessage.Content,
        Messages:   execCtx.messages,
        Steps:      execCtx.step,
        ToolCalls:  toolCallSummaries,
        Usage:      execCtx.totalUsage,
        AgentName:  agentName,
        StartTime:  startTime,
        EndTime:    endTime,
    }, nil
}
```

### 5.3 构造初始消息

```go
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
```

### 5.4 执行工具调用

```go
func (e *Executor) executeToolCalls(
    ctx context.Context,
    toolCalls []llm.ToolCall,
    basePath string,
) []llm.ToolResult {
    return e.toolExecutor.ExecuteAll(ctx, toolCalls)
}
```

**说明**：
- 直接复用 `llm.SafeExecutor.ExecuteAll()`
- 工具执行的安全性、路径验证等由 SafeExecutor 负责

### 5.5 记录工具调用

```go
func (ctx *executionContext) trackToolCall(toolName string) {
    ctx.toolCallCounts[toolName]++
}
```

### 5.6 记录 Token 使用

```go
func (ctx *executionContext) trackUsage(response llm.ChatResponse) {
    if ctx.totalUsage != nil {
        ctx.totalUsage.PromptTokens += response.Usage.PromptTokens
        ctx.totalUsage.CompletionTokens += response.Usage.CompletionTokens
        ctx.totalUsage.TotalTokens += response.Usage.TotalTokens
    }
}
```

---

## 6. 错误处理

### 6.1 错误类型

```go
var (
    // ErrAgentNotFound Agent 不存在
    ErrAgentNotFound = errors.New("agent not found")

    // ErrMaxStepsExceeded 超过最大步骤数
    ErrMaxStepsExceeded = errors.New("max steps exceeded")

    // ErrLLMCallFailed LLM 调用失败
    ErrLLMCallFailed = errors.New("LLM call failed")
)
```

### 6.2 错误处理策略

| 错误场景 | 处理方式 | 是否中断执行 |
|---------|---------|------------|
| Agent 不存在 | 返回 `ErrAgentNotFound` | 是 |
| Tool 执行失败 | 将错误信息作为 tool result 返回给 LLM | 否 |
| LLM 调用失败 | 返回 `ErrLLMCallFailed` | 是 |
| 超过 MaxSteps | 返回 `ErrMaxStepsExceeded` | 是 |
| Context 取消 | 返回 context.Canceled | 是 |

**Tool 执行失败的处理**：
- 不立即中断执行
- 将错误信息通过 tool result 返回给 LLM
- 让 LLM 自主决定如何处理（例如尝试其他工具）

```go
// SafeExecutor 已经处理了工具执行错误
// ToolResult.Content 包含错误信息
// ToolResult.IsError 标记是否出错
```

---

## 7. 日志与监控

### 7.1 关键日志

```go
// 会话开始
klog.Infof("Agent conversation started: agent=%s, steps=%d", agentName, maxSteps)

// 每个执行步骤
klog.Infof("Step %d/%d: agent=%s", step+1, maxSteps, agentName)

// Tool 调用
klog.Infof("Tool call: tool=%s, args=%s, result=%s",
    toolName, args, result.Content)

// 会话结束
klog.Infof("Agent conversation completed: agent=%s, steps=%d, tokens=%d",
    agentName, steps, totalUsage.TotalTokens)
```

### 7.2 监控指标

- **Steps**: 实际执行步骤数
- **ToolCalls**: 调用的工具列表及次数
- **Usage**: Token 使用统计
- **Duration**: 执行时间（EndTime - StartTime）
- **AgentName**: 使用的 Agent 名称

---

## 8. 测试策略

### 8.1 单元测试

**测试文件**：`internal/service/agent/executor_test.go`

**测试用例**：

1. **TestFilterTools**
   - Allow 非空、Deny 空时的过滤
   - Allow 空、Deny 非空时的过滤
   - Allow 和 Deny 都非空时的过滤（deny 优先级）
   - Allow 和 Deny 都空时的过滤

2. **TestBuildMessages**
   - 基本消息构造
   - 带历史记录的消息构造

3. **TestExecuteConversation_AgentNotFound**
   - Agent 不存在时返回错误

4. **TestExecuteConversation_Basic**
   - 基本 LLM 调用（无 tool calls）

5. **TestExecuteConversation_WithToolCalls**
   - 带 tool calls 的 LLM 调用
   - 多轮对话

6. **TestExecuteConversation_MaxSteps**
   - 达到 MaxSteps 时停止

7. **TestExecuteConversation_ToolExecutionFailure**
   - Tool 执行失败时的处理

### 8.2 集成测试

**测试文件**：`internal/service/agent/executor_integration_test.go`

**测试场景**：

1. 使用真实的 Agent 配置
2. 使用真实的 LLM Client（mock 或真实 API）
3. 执行完整的会话流程
4. 验证结果正确性

### 8.3 测试示例

```go
func TestFilterTools(t *testing.T) {
    e := createTestExecutor()

    tests := []struct {
        name        string
        skillPolicy *agents.SkillPolicy
        expected    []string
    }{
        {
            name: "Allow only specific tools",
            skillPolicy: &agents.SkillPolicy{
                Allow: []string{"read_file", "search_files"},
            },
            expected: []string{"read_file", "search_files"},
        },
        {
            name: "Deny specific tools",
            skillPolicy: &agents.SkillPolicy{
                Deny: []string{"execute_bash"},
            },
            expected: allToolsExcept("execute_bash"),
        },
        {
            name: "Deny takes priority over Allow",
            skillPolicy: &agents.SkillPolicy{
                Allow: []string{"read_file", "execute_bash"},
                Deny:  []string{"execute_bash"},
            },
            expected: []string{"read_file"},
        },
        {
            name:        "No filters",
            skillPolicy: &agents.SkillPolicy{},
            expected:    allToolNames(),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            filtered := e.filterTools(tt.skillPolicy)
            names := extractToolNames(filtered)
            assert.Equal(t, tt.expected, names)
        })
    }
}
```

---

## 9. 性能优化

### 9.1 缓存优化

**Default Tools 缓存**：
- `llm.DefaultTools()` 在每次调用时返回新的 slice
- 在 Executor 初始化时缓存默认 tools，避免重复构造

```go
type Executor struct {
    // ...
    defaultTools []llm.Tool  // 缓存默认 tools
}

func NewExecutor(cfg *config.Config) *Executor {
    return &Executor{
        // ...
        defaultTools: llm.DefaultTools(),  // 初始化时缓存
    }
}
```

### 9.2 Tools 过滤优化

- 使用 map 存储 allow 和 deny 列表，O(1) 查找
- 避免在循环中重复构造 map

```go
// 在 filterTools 中只构造一次 map
allowMap := make(map[string]bool, len(skillPolicy.Allow))
for _, toolName := range skillPolicy.Allow {
    allowMap[toolName] = true
}
```

### 9.3 Conversation History 优化

- 限制 history 最大长度（如最近 50 条消息）
- 避免无限制增长导致 token 超限

```go
const maxHistoryLength = 50

if len(history) > maxHistoryLength {
    history = history[len(history)-maxHistoryLength:]
}
```

---

## 10. 安全考虑

### 10.1 Tool 过滤安全

- 必须严格执行 SkillPolicy 过滤
- 未经授权的 tool 绝对不能调用
- 日志记录过滤结果（可用于审计）

### 10.2 Tool 执行安全

- 复用 `llm.SafeExecutor` 的安全机制：
  - 路径验证（ValidatePath）
  - 命令验证（ValidateCommand）
  - 结果长度限制

### 10.3 MaxSteps 安全

- 防止 LLM 无限循环
- 记录每次执行步骤
- 达到 MaxSteps 时强制停止

### 10.4 敏感信息保护

- 日志中不记录 tool arguments 的敏感内容
- 不在错误信息中暴露系统细节
- 日志级别控制（敏感信息只在 DEBUG 级别记录）

---

## 11. 扩展点

### 11.1 自定义 Tools

当前设计使用 `llm.DefaultTools()`，未来可支持：

```go
type Executor struct {
    // ...
    customTools []llm.Tool  // 自定义 tools
}

func (e *Executor) RegisterCustomTool(tool llm.Tool) {
    e.customTools = append(e.customTools, tool)
}
```

### 11.2 Tool Calls 并发执行

当前设计串行执行 tool calls，未来可优化为并发：

```go
func (e *Executor) executeToolCallsConcurrently(
    ctx context.Context,
    toolCalls []llm.ToolCall,
    basePath string,
) []llm.ToolResult {
    // 使用 goroutine 并发执行
}
```

### 11.3 MCP 集成

未来可集成 MCP 获取上下文：

```go
func (e *Executor) enrichSystemPrompt(
    agent *agents.Agent,
    ctx context.Context,
) (string, error) {
    // 根据 Agent.McpPolicy 调用 MCP
    // 将获取的上下文注入到 system prompt
}
```

---

## 12. 实现计划

### Phase 1: 核心功能实现

- [ ] 实现 `ConversationOptions` 和 `ConversationResult` 数据结构
- [ ] 实现 `filterTools()` 方法
- [ ] 实现 `buildMessages()` 方法
- [ ] 实现 `executeToolCalls()` 方法
- [ ] 实现 `ExecuteConversation()` 主方法

### Phase 2: 单元测试

- [ ] 实现 `TestFilterTools`
- [ ] 实现 `TestBuildMessages`
- [ ] 实现 `TestExecuteConversation_AgentNotFound`
- [ ] 实现 `TestExecuteConversation_Basic`
- [ ] 实现 `TestExecuteConversation_WithToolCalls`
- [ ] 实现 `TestExecuteConversation_MaxSteps`

### Phase 3: 集成测试

- [ ] 创建测试 Agent 配置
- [ ] 实现 `TestExecuteConversation_Integration`
- [ ] 验证完整流程

### Phase 4: 优化与文档

- [ ] 性能优化（tools 缓存）
- [ ] 日志完善
- [ ] 使用文档
- [ ] 示例代码

---

## 13. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| Tool 执行失败导致 LLM 无限重试 | 高 | MaxSteps 限制 + 重试计数 |
| Conversation history 过长导致 token 超限 | 中 | 限制 history 最大长度 |
| LLM 返回格式错误 | 中 | 容错处理 + 详细日志 |
| Agent 配置错误（如无效 SkillPolicy） | 低 | 配置校验 + 启动时检查 |

---

## 14. 总结

本设计通过以下核心思想实现 Agent 的 LLM 执行能力：

1. **复用现有组件**：不重复造轮子，复用 LLM Client、ToolExecutor、Agent Manager
2. **职责单一**：Agent Executor 只负责协调，不侵入其他模块
3. **Policy 驱动**：严格遵循 Agent 的 SkillPolicy、RuntimePolicy
4. **渐进式实现**：分阶段实现核心功能、测试、优化
5. **安全优先**：tool 过滤、MaxSteps 限制、安全执行

通过这个设计，Agent 将成为真正可用的智能体，能够带着 tools 发起 LLM 会话，并根据自身的 Policy 约束行为。
