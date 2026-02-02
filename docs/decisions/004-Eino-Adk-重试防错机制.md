 Eino ADK 中 Agent 达到最大运行次数后的重试和兜底机制

---

## 核心问题分析

当 Workflow 中的 Agent 达到 `MaxIterations` 限制时，会返回 `ErrExceedMaxIterations` 错误：

```go
// react.go
var ErrExceedMaxIterations = errors.New("exceeds max iterations")
```

这个错误在 ReAct 循环中触发，导致 Agent 直接退出，推理过程中的内容会丢失。

---

## 一、Eino ADK 内置的重试机制

### 1.1 ChatModel 级别的重试 (`ModelRetryConfig`)

ADK 提供了 `retry_chatmodel.go` 用于配置 ChatModel 调用的重试：

```go
type ModelRetryConfig struct {
    MaxRetries  int                                      // 最大重试次数
    IsRetryAble func(ctx context.Context, err error) bool // 判断错误是否可重试
    BackoffFunc func(ctx context.Context, attempt int) time.Duration // 退避策略
}
```

**关键限制：** `ModelRetryConfig` **只针对 ChatModel 调用失败**（如网络超时），**不解决 MaxIterations 超限** 问题。

---

## 二、MaxIterations 超限的解决方案

### 方案 1：包装 Agent 实现兜底（推荐）

创建一个包装 Agent，捕获 `ErrExceedMaxIterations` 并返回部分结果：

```go
// SafeAgent 包装 Agent，提供兜底处理
type SafeAgent struct {
    inner       adk.Agent
    fallbackFn  func(ctx context.Context, history []adk.Message) (*adk.AgentEvent, error)
}

func (s *SafeAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
    iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
    
    go func() {
        defer gen.Close()
        
        innerIter := s.inner.Run(ctx, input, opts...)
        var history []adk.Message
        
        for {
            event, ok := innerIter.Next()
            if !ok {
                break
            }
            
            // 收集历史消息
            if event.Output != nil && event.Output.MessageOutput != nil {
                if msg, err := event.Output.MessageOutput.GetMessage(); err == nil {
                    history = append(history, msg)
                }
            }
            
            // 捕获 MaxIterations 错误，执行兜底
            if event.Err != nil && errors.Is(event.Err, adk.ErrExceedMaxIterations) {
                fallbackEvent, err := s.fallbackFn(ctx, history)
                if err != nil {
                    gen.Send(&adk.AgentEvent{Err: err})
                } else {
                    gen.Send(fallbackEvent)
                }
                return
            }
            
            gen.Send(event)
        }
    }()
    
    return iter
}

// 使用示例
safeAgent := &SafeAgent{
    inner: originalAgent,
    fallbackFn: func(ctx context.Context, history []adk.Message) (*adk.AgentEvent, error) {
        return &adk.AgentEvent{
            Output: &adk.AgentOutput{
                MessageOutput: &adk.MessageVariant{
                    Message: schema.AssistantMessage(
                        "任务执行时间较长，已达到最大迭代次数。\n\n" +
                        "当前已完成的部分结果：\n" + summarizeHistory(history),
                    ),
                },
            },
            Action: adk.NewExitAction(), // 优雅退出
        }, nil
    },
}
```

---

### 方案 2：利用中断机制保存状态

在接近迭代上限时主动中断，保存状态供恢复：

```go
// 在 Agent 配置中使用中间件监控
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    MaxIterations: 20,
    Middlewares: []adk.AgentMiddleware{
        {
            BeforeChatModel: func(ctx context.Context, state *adk.ChatModelAgentState) error {
                // 通过 State 获取剩余迭代次数
                var remaining int
                compose.ProcessState(ctx, func(ctx context.Context, st *adk.State) error {
                    remaining = st.RemainingIterations
                    return nil
                })
                
                // 接近上限时触发中断
                if remaining <= 3 {
                    return adk.InterruptWithState(ctx, 
                        &ProgressInfo{Message: "即将达到上限"}, 
                        &SavedState{PartialResult: state.Messages},
                    ).Action.Interrupted
                }
                return nil
            },
        },
    },
})
```

---

### 方案 3：Workflow 级别的错误恢复

在 Workflow 中包装 Agent，节点出错时继续执行：

```go
func NewResilientSequentialAgent(ctx context.Context, agents []adk.Agent) (adk.ResumableAgent, error) {
    wrappedAgents := make([]adk.Agent, len(agents))
    
    for i, agent := range agents {
        wrappedAgents[i] = &ResilientAgent{
            inner: agent,
            onError: func(err error) (*adk.AgentEvent, bool) {
                // 如果是 MaxIterations 错误，返回继续信号
                if errors.Is(err, adk.ErrExceedMaxIterations) {
                    return &adk.AgentEvent{
                        Output: &adk.AgentOutput{
                            MessageOutput: &adk.MessageVariant{
                                Message: schema.AssistantMessage("步骤超时，继续下一步..."),
                            },
                        },
                    }, true // 表示已处理，继续执行
                }
                return nil, false
            },
        }
    }
    
    return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
        SubAgents: wrappedAgents,
    })
}
```

---

### 方案 4：增加 MaxIterations + 引导退出（简单方案）

```go
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    MaxIterations: 50, // 增加上限
    Middlewares: []adk.AgentMiddleware{
        {
            AdditionalInstruction: `
                重要提示：如果迭代次数接近上限仍未完成任务，
                请总结当前进度并使用 exit 工具返回结果，
                而不是继续尝试。
            `,
        },
    },
})
```

---

## 三、推荐的分层策略

```
┌─────────────────────────────────────────────────────────────┐
│                     分层容错策略                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Layer 1: ChatModel 重试                                    │
│  ├─ ModelRetryConfig 处理网络/临时错误                        │
│  └─ 默认指数退避 + 抖动                                       │
│                                                             │
│  Layer 2: Agent 迭代兜底                                     │
│  ├─ 包装 Agent 捕获 ErrExceedMaxIterations                   │
│  ├─ 返回部分结果 + Exit Action                               │
│  └─ 历史消息不丢失                                           │
│                                                             │
│  Layer 3: Workflow 节点恢复                                  │
│  ├─ 单节点失败不影响整体流程                                  │
│  └─ 可配置错误处理策略（跳过/重试/兜底）                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 四、生产环境推荐配置

```go
// 完整的生产级配置
func CreateProductionAgent(ctx context.Context, model model.ToolCallingChatModel) (adk.Agent, error) {
    // 1. 基础 Agent
    baseAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:          "production_agent",
        Description:   "Production-ready agent with fallback",
        Model:         model,
        MaxIterations: 30,
        ModelRetryConfig: &adk.ModelRetryConfig{
            MaxRetries: 2,
            IsRetryAble: func(ctx context.Context, err error) bool {
                // 只重试网络相关错误
                var netErr net.Error
                return errors.As(err, &netErr) && netErr.Timeout()
            },
        },
        Middlewares: []adk.AgentMiddleware{
            {
                AdditionalInstruction: `
                    任务执行原则：
                    1. 如果多次尝试后仍无法完成，请总结当前进度并退出
                    2. 优先返回已完成的中间结果
                `,
            },
        },
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 包装为安全 Agent
    return &SafeAgent{
        inner: baseAgent,
        fallbackFn: defaultFallbackFn,
    }, nil
}
```

**总结：** Eino ADK 目前没有内置的 Agent 级别重试机制来处理 MaxIterations 超限，但可以通过 **包装 Agent 模式** 实现优雅的兜底处理，确保推理内容不丢失。