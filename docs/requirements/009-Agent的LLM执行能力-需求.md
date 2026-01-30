# 009-Agent的LLM执行能力-需求.md

## 1. 背景（Why）

当前 Agent 系统已实现 Agent 的定义、加载、路由、注册、热加载等基础设施（需求 006），并具备完善的 LLM 执行器和工具系统（`internal/pkg/llm/`）。

但是，Agent 缺少核心的 LLM 执行能力，导致：

- Agent 只能作为配置容器存储 SystemPrompt、McpPolicy、SkillPolicy、RuntimePolicy
- 无法带着 tools 发起 LLM 会话
- 无法使用 Agent 的 Policy 约束 LLM 行为
- `internal/service/agent/executor.go:135` 有 TODO：`//TODO Agent 如何发起llm对话？`

因此，需要为 Agent 设计并实现 LLM 执行能力，使其成为真正可用的智能体。

---

## 2. 目标（What，必须可验证）

- [ ] 实现 Agent 的 LLM 会话执行能力
- [ ] Agent 能够根据自身 SystemPrompt 发起 LLM 对话
- [ ] Agent 能够根据 SkillPolicy 动态加载和过滤 tools
- [ ] Agent 能够根据 McpPolicy 获取上下文并构造 prompt
- [ ] Agent 能够根据 RuntimePolicy 控制会话执行（maxSteps、riskLevel、确认要求）
- [ ] Agent 能够执行 tool calls 并处理响应
- [ ] 提供 Agent 会话执行接口，供上层服务调用
- [ ] 支持多轮对话（conversation history）
- [ ] 记录 Agent 执行日志（使用的 Agent、调用的 tools、执行步骤）

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不实现 Agent 间协作（multi-agent orchestration）
- [ ] 不实现 Agent 记忆机制（长期记忆存储）
- [ ] 不实现 Agent 的流式响应（streaming response）
- [ ] 不实现 Agent 的成本控制和预算限制
- [ ] 不涉及前端 UI 实现
- [ ] 不实现 Agent 的性能优化（如工具调用缓存）

---

## 4. 使用场景 / 用户路径

### 场景 1：仓库分析 Agent 执行

```
用户发起仓库分析请求
    ↓
Agent Router 选择 repo-analysis-agent
    ↓
Agent 构造 system prompt（来自 Agent.SystemPrompt）
    ↓
Agent 根据 SkillPolicy 过滤可用 tools（只允许 search_files, read_file 等）
    ↓
Agent 调用 LLM，携带 filtered tools
    ↓
LLM 返回 tool calls（例如调用 search_files）
    ↓
Agent 使用 ToolExecutor 执行 tools
    ↓
Agent 将 tool results 返回给 LLM
    ↓
LLM 继续思考或返回最终结果
    ↓
Agent 返回分析结果给用户
```

### 场景 2：显式指定 Agent

```
用户通过 API 调用指定 agent_name="code-diagnosis-agent"
    ↓
Agent Router 直接返回指定的 Agent
    ↓
Agent 使用 code-diagnosis-agent 的 SystemPrompt 和 Policy
    ↓
Agent 执行 LLM 会话
```

---

## 5. 功能需求清单（Checklist）

### 5.1 Agent LLM 执行接口

- [ ] 实现 `ExecuteConversation(ctx, agentName, userMessage, options) (result, error)` 接口
- [ ] 支持传入 conversation history（多轮对话）
- [ ] 支持执行配置（maxSteps、temperature 等）

### 5.2 Tools 动态加载与过滤

- [ ] 根据 Agent.SkillPolicy 过滤系统默认 tools（`llm.DefaultTools()`）
- [ ] 过滤规则：
  - 如果 SkillPolicy.Allow 非空，只保留在 Allow 列表中的 tools
  - 移除在 SkillPolicy.Deny 列表中的 tools（deny 优先级高于 allow）
- [ ] 支持自定义 tools 注册（除默认 tools 外的工具）

### 5.3 System Prompt 构造

- [ ] 使用 Agent.SystemPrompt 作为 system message
- [ ] 支持 MCP Policy 动态注入上下文到 system prompt（如果需要）
- [ ] 保留 system prompt 的格式和换行

### 5.4 LLM 会话执行

- [ ] 构造 ChatRequest：
  - messages = [system_message, user_message, history]
  - tools = 过滤后的 tools
  - tool_choice = "auto"（让 LLM 自主决定是否调用工具）
- [ ] 调用 `llm.Client.Chat(ctx, request)`
- [ ] 处理 LLM 响应：
  - 如果返回 tool_calls，执行工具调用
  - 如果返回 content，返回给用户

### 5.5 Tool Calls 执行

- [ ] 使用 `llm.SafeExecutor` 执行 tool calls
- [ ] 将 tool results 转换为 assistant tool messages
- [ ] 将 tool results 添加到 conversation history
- [ ] 重新调用 LLM，携带 tool results

### 5.6 多轮对话与步骤控制

- [ ] 维护 conversation history（system + user + assistant + tool messages）
- [ ] 根据 RuntimePolicy.MaxSteps 控制最大执行轮数
- [ ] 达到 MaxSteps 时停止执行并返回

### 5.7 执行日志与监控

- [ ] 记录使用的 Agent 名称
- [ ] 记录执行的 tools 及调用次数
- [ ] 记录执行步骤数
- [ ] 记录执行时间
- [ ] 记录 LLM token 使用情况

### 5.8 错误处理

- [ ] Agent 不存在时返回 `ErrAgentNotFound`
- [ ] Tool 执行失败时，将错误信息返回给 LLM（让 LLM 自主处理）
- [ ] LLM 调用失败时返回错误
- [ ] 超过 MaxSteps 时返回错误

---

## 6. 约束条件

### 6.1 技术约束

- 必须复用现有的 `llm.Client` 和 `llm.SafeExecutor`
- 必须复用现有的 `agents.Manager` 和 `agents.Router`
- 不得修改 Agent 定义结构（保持向后兼容）
- 必须支持 Go 1.21+

### 6.2 架构约束

- Agent Executor 应作为独立的服务层，不与 agents 包耦合
- LLM 执行逻辑应在 `internal/service/agent/executor.go` 中实现
- 不得在 `internal/pkg/agents/` 包中添加 LLM 相关代码（保持职责单一）

### 6.3 安全约束

- 必须严格执行 SkillPolicy 过滤（未经授权的 tool 不可调用）
- 必须严格执行 RuntimePolicy.MaxSteps 限制
- 必须正确处理 tool 执行失败（不泄露敏感信息）
- 必须验证 tool arguments 的有效性

### 6.4 性能约束

- Tool 过滤应在 1ms 内完成（假设有 100 个 tools）
- 单轮 LLM 调用应在 10s 内完成（取决于 LLM 响应速度）
- MaxSteps=10 的完整会话应在 60s 内完成

---

## 7. 可修改 / 不可修改项

- ❌ 不可修改：
  - Agent 定义结构（`internal/pkg/agents/agent.go`）
  - LLM Client 接口（`internal/pkg/llm/client.go`）
  - ToolExecutor 接口（`internal/pkg/llm/executor.go`）
  - SkillPolicy 过滤规则逻辑

- ✅ 可调整：
  - Conversation history 的维护方式
  - Tool calls 的执行顺序（并行 vs 串行）
  - 错误处理方式（是否在 tool 执行失败时立即终止）
  - 日志记录的详细程度

---

## 8. 接口与数据约定

### 8.1 Agent Executor 执行接口

```go
// ConversationOptions 会话执行配置
type ConversationOptions struct {
    ConversationHistory []llm.ChatMessage // 历史消息（多轮对话）
    MaxSteps           int               // 最大执行步骤（覆盖 Agent.RuntimePolicy.MaxSteps）
    Temperature         float64           // 温度参数
    BasePath           string            // 基础路径（用于 tool 执行）
}

// ConversationResult 会话执行结果
type ConversationResult struct {
    Content      string            // 最终响应内容
    Messages     []llm.ChatMessage // 完整对话历史
    Steps        int               // 实际执行步骤数
    ToolCalls    []ToolCallSummary // 调用的 tools
    Usage        *LLMUsage         // Token 使用统计
    AgentName    string            // 使用的 Agent 名称
}

// ToolCallSummary 工具调用摘要
type ToolCallSummary struct {
    ToolName string `json:"tool_name"`
    Count    int    `json:"count"`
}

// LLMUsage LLM 使用统计
type LLMUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// ExecuteConversation 执行 Agent 会话
func (e *Executor) ExecuteConversation(
    ctx context.Context,
    agentName string,
    userMessage string,
    options *ConversationOptions,
) (*ConversationResult, error)
```

### 8.2 执行流程伪代码

```go
func ExecuteConversation(ctx, agentName, userMessage, options) (*ConversationResult, error) {
    // 1. 获取 Agent
    agent, err := e.manager.Get(agentName)
    if err != nil {
        return nil, ErrAgentNotFound
    }

    // 2. 过滤可用 tools
    availableTools := e.filterTools(agent.SkillPolicy)

    // 3. 构造 conversation history
    messages := []llm.ChatMessage{
        {Role: "system", Content: agent.SystemPrompt},
    }
    if options.ConversationHistory != nil {
        messages = append(messages, options.ConversationHistory...)
    }
    messages = append(messages, llm.ChatMessage{Role: "user", Content: userMessage})

    // 4. 执行多轮对话
    steps := 0
    maxSteps := agent.RuntimePolicy.MaxSteps
    if options.MaxSteps > 0 {
        maxSteps = options.MaxSteps
    }

    for steps < maxSteps {
        // 5. 调用 LLM
        request := llm.ChatRequest{
            Model:       e.llmClient.model,
            Messages:    messages,
            Tools:       availableTools,
            ToolChoice:  "auto",
            Temperature: options.Temperature,
        }
        response, err := e.llmClient.Chat(ctx, request)
        if err != nil {
            return nil, err
        }

        // 6. 处理响应
        assistantMessage := llm.ChatMessage{
            Role: "assistant",
            Content: response.Choices[0].Message.Content,
            ToolCalls: response.Choices[0].Message.ToolCalls,
        }
        messages = append(messages, assistantMessage)

        // 7. 如果没有 tool_calls，结束对话
        if len(assistantMessage.ToolCalls) == 0 {
            break
        }

        // 8. 执行 tool calls
        toolResults := e.toolExecutor.ExecuteAll(ctx, assistantMessage.ToolCalls)

        // 9. 将 tool results 转换为 messages
        for i, result := range toolResults {
            messages = append(messages, llm.ChatMessage{
                Role: "tool",
                Content: result.Content,
                ToolCallID: assistantMessage.ToolCalls[i].ID,
            })
        }

        steps++
    }

    // 10. 返回结果
    return &ConversationResult{
        Content:   assistantMessage.Content,
        Messages:  messages,
        Steps:     steps,
        AgentName: agentName,
    }, nil
}
```

---

## 9. 验收标准

### 9.1 功能验收

- [ ] 如果 Agent 不存在，ExecuteConversation 应返回 `ErrAgentNotFound`
- [ ] 如果 Agent.SkillPolicy.Allow 非空，只有允许的 tools 应出现在 LLM 请求中
- [ ] 如果 Tool 在 SkillPolicy.Deny 列表中，该 tool 不应出现在 LLM 请求中
- [ ] 如果 LLM 返回 tool_calls，Executor 应执行这些 tools
- [ ] 如果 tool 执行成功，结果应返回给 LLM
- [ ] 如果 tool 执行失败，错误信息应返回给 LLM
- [ ] 如果达到 MaxSteps，Executor 应停止执行并返回当前结果
- [ ] ConversationHistory 应被正确传递给 LLM
- [ ] 最终响应内容应从 LLM 的 assistant message 中提取
- [ ] 应记录使用的 Agent 名称

### 9.2 集成验收

- [ ] 应能与现有的 `agents.Manager` 集成
- [ ] 应能与现有的 `llm.Client` 集成
- [ ] 应能与现有的 `llm.SafeExecutor` 集成
- [ ] 应能执行 `llm.DefaultTools()` 中的所有工具（如果被 SkillPolicy 允许）
- [ ] 应能解析和执行 tool calls

### 9.3 安全验收

- [ ] 未经授权的 tool 不应被调用
- [ ] MaxSteps 应正确限制执行步骤
- [ ] Tool arguments 验证应正确执行
- [ ] 敏感信息不应泄露在日志中

### 9.4 性能验收

- [ ] Tool 过滤应在 1ms 内完成（100 tools）
- [ ] 单轮 LLM 调用应在 10s 内完成
- [ ] MaxSteps=10 的完整会话应在 60s 内完成

---

## 10. 风险与已知不确定点

| 风险点 | 说明 | 处理方式 |
|-------|------|---------|
| Tool 执行失败 | LLM 调用的 tool 执行失败时如何处理 | 将错误信息返回给 LLM，让 LLM 自主决定是否继续 |
| 无限循环 | LLM 重复调用同一个 tool | 通过 MaxSteps 限制，记录警告日志 |
| Tool 并发执行 | LLM 返回多个 tool calls 时是否并发执行 | 先串行执行，后续可优化为并发 |
| Memory 泄漏 | Conversation history 不断增长 | 限制 history 最大长度（如最近 50 条消息） |
| Token 超限 | Conversation history 过长导致 LLM 无法处理 | 记录警告，建议用户重新开始对话 |

---

## 11. 与现有系统的关系

### 11.1 与 Agent Manager 的关系

```
                    +------------------+
                    |  Agent Manager   |
                    |  - Get(agent)    |
                    |  - SelectAgent() |
                    +------------------+
                              |
                              | 提供 Agent 定义
                              ↓
+------------------+   +------------------+
|   Agent Executor | ← |      Agent       |
| - ExecuteConv()  |   | - SystemPrompt  |
| - FilterTools()  |   | - SkillPolicy   |
| - ExecuteTools() |  | - RuntimePolicy |
+------------------+   +------------------+
         |
         | 执行 tools
         ↓
+------------------+
|  LLM Client      |
| - Chat()         |
+------------------+
         |
         | tool calls
         ↓
+------------------+
| Tool Executor    |
| - ExecuteAll()   |
+------------------+
```

### 11.2 与现有代码的关系

- 修改文件：
  - `internal/service/agent/executor.go`（新增 ExecuteConversation 方法）

- 不修改文件：
  - `internal/pkg/agents/`（Agent 定义不变）
  - `internal/pkg/llm/client.go`（LLM Client 不变）
  - `internal/pkg/llm/executor.go`（Tool Executor 不变）

---

## 12. 交付物

- [ ] Agent LLM 执行接口实现
- [ ] Tools 动态过滤逻辑
- [ ] Conversation history 管理
- [ ] Tool calls 执行与结果处理
- [ ] 执行日志与监控
- [ ] 单元测试
- [ ] 集成测试（使用示例 Agent）
- [ ] 使用文档

---

## 13. 示例使用代码

### 13.1 基本使用

```go
// 创建 Executor
executor := agent.NewExecutor(cfg)

// 执行会话
result, err := executor.ExecuteConversation(
    ctx,
    "repo-analysis-agent",
    "请分析这个仓库的结构",
    &agent.ConversationOptions{
        MaxSteps:   10,
        BasePath:   "/path/to/repo",
    },
)

if err != nil {
    log.Fatalf("执行失败: %v", err)
}

fmt.Printf("响应: %s\n", result.Content)
fmt.Printf("执行步骤: %d\n", result.Steps)
```

### 13.2 多轮对话

```go
// 第一轮
result1, _ := executor.ExecuteConversation(ctx, "my-agent", "列出所有 Go 文件", nil)

// 第二轮（带历史）
history := append(result1.Messages)
result2, _ := executor.ExecuteConversation(
    ctx,
    "my-agent",
    "读取 main.go 的内容",
    &agent.ConversationOptions{
        ConversationHistory: history,
    },
)
```

---

## 14. 后续扩展方向（非本次需求）

- [ ] 支持 Agent 间协作（multi-agent orchestration）
- [ ] 支持 Agent 长期记忆机制
- [ ] 支持 Agent 流式响应
- [ ] 支持 Agent 成本控制和预算限制
- [ ] 支持 Tool 调用并发执行优化
- [ ] 支持 Agent 性能监控和分析
