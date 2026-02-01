# ADK Agents 管理模块

基于 YAML 配置的 ADK Agent 管理模块，支持热加载和动态扩展。

## 功能特性

- **YAML 配置化**：Agent 定义通过 YAML 文件配置，无需修改代码
- **热加载支持**：配置文件变更自动加载，无需重启服务
- **实例缓存**：ADK Agent 实例自动缓存，避免重复创建
- **灵活扩展**：支持自定义 ModelProvider 和 ToolProvider

## 快速开始

### 1. 创建 YAML 配置

```yaml
# agents/my-agent.yaml
name: MyAgent
description: 我的 Agent

instruction: |
  你是一个有用的助手，可以帮助用户完成任务。

tools:
  - list_dir
  - read_file

maxIterations: 10
```

### 2. 创建 Manager

```go
import "github.com/opendeepwiki/backend/internal/pkg/adkagents"

// 创建 Provider
modelProvider := adkagents.NewSimpleModelProvider(chatModel)
toolProvider := adkagents.NewSimpleToolProvider()
toolProvider.RegisterTool("list_dir", listDirTool)
toolProvider.RegisterTool("read_file", readFileTool)

// 创建 Manager
config := &adkagents.Config{
    Dir:            "./agents",
    AutoReload:     true,
    ReloadInterval: 5 * time.Second,
    ModelProvider:  modelProvider,
    ToolProvider:   toolProvider,
}

manager, err := adkagents.NewManager(config)
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()
```

### 3. 获取 Agent

```go
agent, err := manager.GetAgent("MyAgent")
if err != nil {
    log.Fatal(err)
}

// 使用 ADK Agent
result, err := agent.Run(ctx, input)
```

## 配置说明

### Manager 配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Dir | string | 否 | Agent 配置目录，默认 "./agents" |
| AutoReload | bool | 否 | 是否启用热加载，默认 true |
| ReloadInterval | time.Duration | 否 | 热加载检查间隔，默认 5s |
| ModelProvider | ModelProvider | 是 | 模型提供者 |
| ToolProvider | ToolProvider | 是 | 工具提供者 |

### Agent YAML 配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Agent 唯一标识 |
| description | string | 是 | Agent 描述 |
| model | string | 否 | 模型名称，空则使用默认 |
| instruction | string | 是 | System Prompt |
| tools | []string | 否 | 工具名称列表 |
| maxIterations | int | 是 | 最大迭代次数 |
| exit | object | 否 | 退出条件配置 |

## 环境变量

- `ADK_AGENTS_DIR`: 指定 Agent 配置目录
- `ADK_AGENTS_AUTO_RELOAD`: 是否启用热加载（true/false）

## 接口说明

### Manager 接口

```go
type Manager struct {
    // 获取指定名称的 ADK Agent 实例
    GetAgent(name string) (adk.Agent, error)
    
    // 列出所有 Agent 定义
    List() []*AgentDefinition
    
    // 重新加载指定 Agent
    Reload(name string) error
    
    // 停止 Manager，关闭文件监听
    Stop()
}
```

### Provider 接口

```go
// ModelProvider 模型提供者接口
type ModelProvider interface {
    GetModel(name string) (model.ToolCallingChatModel, error)
    DefaultModel() model.ToolCallingChatModel
}

// ToolProvider 工具提供者接口
type ToolProvider interface {
    GetTool(name string) (tool.BaseTool, error)
    ListTools() []string
}
```

## 错误处理

| 错误 | 说明 |
|------|------|
| ErrAgentNotFound | Agent 不存在 |
| ErrInvalidConfig | 配置文件无效 |
| ErrInvalidName | Agent 名称格式无效 |
| ErrToolNotFound | 工具不存在 |
| ErrModelNotFound | 模型不存在 |

## 注意事项

1. Agent 名称必须唯一，由字母、数字、连字符、下划线组成
2. 工具不存在时会记录警告并跳过，不会阻断 Agent 加载
3. 模型不存在时会使用默认模型
4. 配置文件变更后，缓存会自动失效，下次 GetAgent 会重新创建实例
