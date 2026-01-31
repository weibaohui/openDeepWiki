# Eino ADK 原生模式 - 仓库文档生成服务

基于 Eino ADK (Agent Development Kit) **原生 API** 实现的代码仓库文档生成系统。

## 与自定义实现的区别

本实现严格遵循 Eino ADK 的原生 API：

| 组件 | 原生 ADK API | 说明 |
|------|-------------|------|
| ChatModelAgent | `adk.NewChatModelAgent()` | 使用原生 ChatModelAgent |
| SequentialAgent | `adk.NewSequentialAgent()` | 使用原生 SequentialAgent |
| Runner | `adk.NewRunner()` | 使用原生 Runner |
| Agent 接口 | `adk.Agent` | 实现标准 Agent 接口 |
| ToolsConfig | `adk.ToolsConfig` | 使用原生工具配置 |

## 架构设计

```
RepoDocWorkflow (使用原生 adk.SequentialAgent)
    │
    ├── adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
    │       SubAgents: []adk.Agent{
    │           ├── adk.NewChatModelAgent() // RepoInitializer
    │           ├── adk.NewChatModelAgent() // Architect  
    │           ├── adk.NewChatModelAgent() // Explorer
    │           ├── adk.NewChatModelAgent() // Writer
    │           └── adk.NewChatModelAgent() // Editor
    │       }
    │   })
    │
    └── adk.NewRunner(ctx, adk.RunnerConfig{Agent: sequentialAgent})
            └── runner.Run(ctx, messages)
```

## Agent 职责

| Agent 名称 | 原生创建方式 | 职责描述 | 工具 |
|------------|-------------|----------|------|
| **RepoInitializer** | `adk.NewChatModelAgent(...)` | 克隆仓库、获取目录结构 | git_clone, list_dir |
| **Architect** | `adk.NewChatModelAgent(...)` | 分析仓库类型、生成文档大纲 | 无（仅 LLM） |
| **Explorer** | `adk.NewChatModelAgent(...)` | 深度探索代码结构 | read_file, search_files |
| **Writer** | `adk.NewChatModelAgent(...)` | 生成文档内容 | read_file |
| **Editor** | `adk.NewChatModelAgent(...)` | 组装最终文档 | 无（仅 LLM） |

## 核心代码示例

### 创建 ChatModelAgent（原生方式）

```go
import "github.com/cloudwego/eino/adk"

agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "RepoInitializer",
    Description: "仓库初始化专员",
    Instruction: `你是仓库初始化专员...`,
    Model:       chatModel,  // model.ToolCallingChatModel
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{
                NewGitCloneToolWrapper(basePath),
                NewListDirToolWrapper(basePath),
            },
        },
    },
    MaxIterations: 10,
})
```

### 创建 SequentialAgent（原生方式）

```go
sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
    Name:        "RepoDocSequentialAgent",
    Description: "仓库文档生成顺序执行 Agent",
    SubAgents: []adk.Agent{
        initializer,  // adk.Agent 接口
        architect,
        explorer,
        writer,
        editor,
    },
})
```

### 执行 Workflow（原生方式）

```go
// 创建 Runner
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent: sequentialAgent,
})

// 设置会话值
adk.AddSessionValue(ctx, "repo_url", repoURL)
adk.AddSessionValue(ctx, "base_path", basePath)

// 执行
iter := runner.Run(ctx, []adk.Message{
    {Role: schema.User, Content: "请分析这个仓库..."},
})

// 处理事件
for {
    event, ok := iter.Next()
    if !ok { break }
    
    if event.Err != nil {
        log.Fatal(event.Err)
    }
    
    if event.Output != nil {
        fmt.Println(event.Output.MessageOutput.Message.Content)
    }
    
    // 检查退出
    if event.Action != nil && event.Action.Exit {
        break
    }
}
```

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

    log.Printf("文档生成成功！共 %d 个小节\n", result.SectionsCount)
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
        fmt.Printf("✓ 全部完成！\n")
    case adk.WorkflowStatusError:
        fmt.Printf("✗ [%s] 出错: %v\n", event.AgentName, event.Error)
    }
}
```

## 文件结构

```
adk/
├── README.md           # 本文档
├── types.go            # 类型定义（AgentRole、WorkflowInput/Output 等）
├── state.go            # 状态管理器（StateManager）
├── wrapper.go          # 辅助函数
├── agents.go           # 使用原生 adk.NewChatModelAgent 创建 Agent
├── workflow.go         # 使用原生 adk.SequentialAgent 和 adk.Runner
├── service.go          # 对外服务接口
└── example_test.go     # 使用示例和测试
```

## 关键改进

### 1. 使用原生 ChatModelAgent

```go
// 原生方式
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "AgentName",
    Description: "Agent Description",
    Instruction: "System Prompt...",
    Model:       chatModel,
    ToolsConfig: adk.ToolsConfig{...},
    MaxIterations: 10,
})
```

相比自定义实现：
- 支持 ReAct 模式（自动 Reasoning + Acting）
- 支持 `ReturnDirectly` 工具配置
- 支持 `MaxIterations` 限制
- 支持 `ModelRetryConfig` 重试配置

### 2. 使用原生 SequentialAgent

```go
// 原生方式
sequentialAgent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
    Name:        "WorkflowName",
    Description: "Workflow Description",
    SubAgents:   []adk.Agent{agent1, agent2, ...},
})
```

返回的是 `adk.ResumableAgent` 接口，支持：
- 顺序执行
- 断点续跑（Resume）
- 检查点（Checkpoint）

### 3. 使用原生 Runner

```go
// 原生方式
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
iter := runner.Run(ctx, messages)
```

支持：
- `Query()` - 简单查询
- `Run()` - 完整对话
- `Resume()` - 断点续跑

## Eino ADK 官方参考

- [Eino ADK Overview](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/)
- [ChatModelAgent](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
- [Workflow Agents](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/workflow/)
- [GitHub Examples](https://github.com/cloudwego/eino-examples/tree/main/adk)

## 注意事项

1. **工具包装器**：由于原工具有不同的接口，需要创建包装器实现 `tool.BaseTool` 接口
2. **会话值**：使用 `adk.AddSessionValue()` 在 Agent 之间传递上下文
3. **退出信号**：`event.Action.Exit` 是 `bool` 类型，不是指针
4. **工具配置**：`ToolsConfig` 嵌入了 `compose.ToolsNodeConfig`，需要通过它设置工具

## 与原代码的兼容性

### 复用的组件

- `einodoc.RepoDocState` - 状态管理
- `einodoc.RepoDocResult` - 结果结构
- `einodoc.Chapter/Section` - 文档大纲结构
- `tools.GitCloneTool` - Git 克隆工具（通过包装器）
- `tools.ListDirTool` - 目录列表工具（通过包装器）
- `tools.ReadFileTool` - 文件读取工具（通过包装器）
- `einodoc.NewLLMChatModel` - LLM 模型创建
