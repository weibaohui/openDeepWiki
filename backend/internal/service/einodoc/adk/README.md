# Eino ADK 模式 - 仓库文档生成服务

基于 Eino ADK (Agent Development Kit) 的 SequentialAgent 模式实现的代码仓库文档生成系统。

## 架构设计

### 核心概念

```
┌─────────────────────────────────────────────────────────────────┐
│                    RepoDocSequentialWorkflow                     │
│                        (SequentialAgent)                         │
└──────────────────────────────┬──────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
        ▼                      ▼                      ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ RepoInitializer│ -> │   Architect   │ -> │   Explorer    │
│   (Agent 1)   │    │   (Agent 2)   │    │   (Agent 3)   │
└───────────────┘    └───────────────┘    └───────────────┘
                               │                      │
                               ▼                      ▼
                       ┌───────────────┐    ┌───────────────┐
                       │    Writer     │ -> │    Editor     │
                       │   (Agent 4)   │    │   (Agent 5)   │
                       └───────────────┘    └───────────────┘
```

### Agent 职责

| Agent 名称 | 职责描述 | 对应原流程 |
|------------|----------|------------|
| **RepoInitializer** | 克隆仓库、获取目录结构 | Step 1: Clone & Read Tree |
| **Architect** | 分析仓库类型、生成文档大纲 | Step 2: Pre-read Analysis + Step 3: Generate Outline |
| **Explorer** | 深度探索代码结构、读取关键文件 | （新增）深度分析阶段 |
| **Writer** | 为每个小节生成文档内容 | Step 4: Generate Section Content |
| **Editor** | 组装最终文档、优化格式 | Step 5: Finalize Document |

## 与原有 Chain 模式的对比

### Chain 模式 (原有实现)

```go
chain := compose.NewChain[WorkflowInput, WorkflowOutput]()
chain.AppendLambda(step1)
chain.AppendLambda(step2)
...
chain.Compile(ctx)
```

特点：
- 基于函数式编程的 Lambda 组合
- 适合简单的线性流程
- 代码紧凑，但扩展性有限

### ADK SequentialAgent 模式 (新实现)

```go
agents := []Agent{
    CreateRepoInitializerAgent(state),
    CreateArchitectAgent(state),
    CreateExplorerAgent(state),
    CreateWriterAgent(state),
    CreateEditorAgent(state),
}

sequentialAgent := NewSequentialAgent(ctx, &SequentialAgentConfig{
    SubAgents: agents,
})
```

特点：
- 基于 Agent 的抽象，每个 Agent 有明确的角色和职责
- 支持更复杂的协作模式（Sequential、Parallel、Loop、Supervisor）
- 易于扩展和替换单个 Agent
- 更好的可观测性（每个 Agent 的输入输出清晰）

## 使用方式

### 基本使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/opendeepwiki/backend/internal/service/einodoc/adk"
)

func main() {
    // 配置 LLM
    llmCfg := &adk.LLMConfig{
        APIKey:    "your-api-key",
        BaseURL:   "https://api.openai.com/v1",
        Model:     "gpt-4o",
        MaxTokens: 4000,
    }

    // 创建服务
    service, err := adk.NewADKRepoDocService("/tmp/repos", llmCfg)
    if err != nil {
        log.Fatal(err)
    }

    // 解析仓库
    ctx := context.Background()
    result, err := service.ParseRepo(ctx, "https://github.com/example/project.git")
    if err != nil {
        log.Fatal(err)
    }

    // 输出结果
    log.Printf("文档生成成功！共 %d 个小节\n", result.SectionsCount)
    log.Printf("文档长度: %d 字符\n", len(result.Document))
}
```

### 带进度反馈的解析

```go
progressCh, err := service.ParseRepoWithProgress(ctx, repoURL)
if err != nil {
    log.Fatal(err)
}

for event := range progressCh {
    switch event.Status {
    case adk.WorkflowStatusCompleted:
        fmt.Printf("✓ [%s] 完成\n", event.AgentName)
    case adk.WorkflowStatusFinished:
        fmt.Printf("✓ 全部完成！共 %d 个小节\n", event.Result.SectionsCount)
    case adk.WorkflowStatusError:
        fmt.Printf("✗ [%s] 出错: %v\n", event.AgentName, event.Error)
    }
}
```

## 文件结构

```
adk/
├── README.md           # 本文档
├── types.go            # 类型定义（WorkflowInput/Output, AgentRole 等）
├── state.go            # 状态管理器（StateManager）
├── wrapper.go          # Agent 包装器（SequentialAgent, Runner 等）
├── agents.go           # 各个子 Agent 的实现
├── workflow.go         # Sequential Workflow 主逻辑
├── service.go          # 对外服务接口
└── example_test.go     # 使用示例和测试
```

## 扩展开发

### 添加新的 Agent

1. 在 `types.go` 中定义新的 Agent 角色：

```go
const (
    AgentNewAgent = "NewAgent"
)

AgentRoles[AgentNewAgent] = AgentRole{
    Name:        AgentNewAgent,
    Description: "新 Agent 描述",
    Instruction: "Agent 的系统指令...",
}
```

2. 在 `AgentFactory` 中添加创建方法：

```go
func (f *AgentFactory) CreateNewAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
    role := AgentRoles[AgentNewAgent]
    return &ChatModelAgentWrapper{
        name:        role.Name,
        description: role.Description,
        state:       state,
        doExecute:   f.executeNewAgent,
    }, nil
}

func (f *AgentFactory) executeNewAgent(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
    // 实现 Agent 逻辑
    return &schema.Message{
        Role:    schema.Assistant,
        Content: "执行结果",
    }, nil
}
```

3. 在 `workflow.go` 的 `Build` 方法中添加新 Agent：

```go
newAgent, err := factory.CreateNewAgent(w.state)
agents = append(agents, newAgent)
```

### 使用 ParallelAgent 并行处理

对于可以并行处理的章节，可以使用 ParallelAgent：

```go
// 创建多个 Writer Agent，每个负责一个章节
writerAgents := make([]Agent, len(outline))
for i, chapter := range outline {
    writerAgents[i] = factory.CreateChapterWriterAgent(state, i, chapter)
}

// 使用 ParallelAgent 并行执行
parallelWriter := NewParallelAgent(ctx, &ParallelAgentConfig{
    Name:        "ParallelWriters",
    Description: "并行撰写各章节",
    SubAgents:   writerAgents,
})
```

## 与原代码的兼容性

### 复用的组件

ADK 模式复用了原 Chain 模式下的以下组件：

- `einodoc.RepoDocState` - 状态管理（通过 StateManager 包装）
- `einodoc.RepoDocResult` - 结果结构
- `einodoc.Chapter/Section` - 文档大纲结构
- `tools.GitCloneTool` - Git 克隆工具
- `tools.ListDirTool` - 目录列表工具
- `tools.ReadFileTool` - 文件读取工具
- `einodoc.NewLLMChatModel` - LLM 模型创建

### 服务接口对比

| 功能 | Chain 模式 | ADK 模式 |
|------|-----------|----------|
| 创建服务 | `NewEinoRepoDocService(...)` | `NewADKRepoDocService(...)` |
| 解析仓库 | `ParseRepo(ctx, repoURL)` | `ParseRepo(ctx, repoURL)` |
| 带进度解析 | 不支持 | `ParseRepoWithProgress(...)` |
| 获取信息 | `GetCallbacks()` | `GetWorkflowInfo()` |

## 注意事项

1. **API Key**: 需要配置有效的 OpenAI API Key 或其他兼容的 LLM 服务
2. **存储空间**: 仓库克隆到本地需要足够的磁盘空间
3. **网络连接**: 需要访问 Git 仓库和 LLM API
4. **超时控制**: 建议为 `ParseRepo` 设置合理的超时时间

## 参考资料

- [Eino ADK 官方文档](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/)
- [SequentialAgent 文档](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/workflow/)
- 原 Chain 模式实现: `backend/internal/service/einodoc/workflow.go`
