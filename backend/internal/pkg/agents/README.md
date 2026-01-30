# Agent 管理与加载

本包提供 Agent（智能体）的管理和加载功能，支持通过 YAML/JSON 配置文件定义 Agent，并在运行时动态加载和路由。

## 核心概念

### Agent

Agent 是一个会话级执行单元，封装了对大模型的角色定义、能力边界和行为策略：

- **System Prompt**: 定义 Agent 的角色和行为约束
- **MCP Policy**: 定义可用的 MCP（上下文获取能力）及调用限制
- **Skill Policy**: 定义可用的 Skills（执行能力）
- **Runtime Policy**: 定义运行时约束（风险等级、最大步骤、确认要求）

### 组件架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Agent Manager                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Parser    │  │   Loader    │  │      Registry       │ │
│  │ (配置解析)   │  │ (加载器)     │  │    (注册中心)        │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐                          │
│  │   Router    │  │   Watcher   │                          │
│  │  (路由器)    │  │  (热加载)    │                          │
│  └─────────────┘  └─────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

## 使用方法

### 1. 创建 Manager

```go
import "github.com/opendeepwiki/backend/internal/pkg/agents"

// 使用默认配置
manager, err := agents.NewManager(nil)
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()

// 或使用自定义配置
config := &agents.Config{
    Dir:            "./agents",
    AutoReload:     true,
    ReloadInterval: 5 * time.Second,
    DefaultAgent:   "default-agent",
    Routes: map[string]string{
        "diagnose": "diagnose-agent",
        "ops":      "ops-agent",
    },
}
manager, err := agents.NewManager(config)
```

### 2. 选择 Agent

```go
// 根据上下文选择 Agent
ctx := agents.RouterContext{
    EntryPoint: "diagnose", // 用户入口
}

agent, err := manager.SelectAgent(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Selected agent: %s\n", agent.Name)
fmt.Printf("System Prompt: %s\n", agent.SystemPrompt)
```

### 3. 检查 Skill 权限

```go
// 检查 Agent 是否允许使用某个 Skill
if agent.IsSkillAllowed("restart_pod") {
    // 允许使用
} else {
    // 禁止使用
}
```

## Agent 配置格式

### YAML 格式示例

```yaml
name: diagnose-agent
version: v1
description: Kubernetes 服务诊断 Agent

systemPrompt: |
  你是一个谨慎的系统诊断专家。
  你的目标是分析问题根因，而不是立即修改系统。
  在没有充分证据前，禁止执行任何写操作。

mcp:
  allowed:
    - cluster_state
    - pod_logs
    - metrics
  maxCalls: 5

skills:
  allow:
    - search_logs
    - analyze_logs
  deny:
    - restart_pod
    - delete_resource

policies:
  riskLevel: read
  maxSteps: 6
  requireConfirmation: false
```

### 字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | 是 | Agent 唯一标识，小写字母、数字、连字符 |
| version | string | 是 | 版本号，格式 v1, v1.0, v1.0.0 |
| description | string | 是 | Agent 描述，最多 1024 字符 |
| systemPrompt | string | 是 | System Prompt 内容 |
| mcp.allowed | []string | 否 | 允许的 MCP 列表 |
| mcp.maxCalls | int | 否 | MCP 最大调用次数 |
| skills.allow | []string | 否 | 允许的 Skills 列表 |
| skills.deny | []string | 否 | 禁止的 Skills 列表（优先级高于 allow）|
| policies.riskLevel | string | 否 | 风险等级：read / write / admin |
| policies.maxSteps | int | 否 | 最大执行步骤数 |
| policies.requireConfirmation | bool | 否 | 是否需要确认 |

## 路由规则

Router 按照以下优先级选择 Agent：

1. **显式指定**: `RouterContext.AgentName`
2. **入口路由**: 根据 `RouterContext.EntryPoint` 匹配路由规则
3. **默认 Agent**: 使用配置的默认 Agent

```go
// 显式指定
ctx := agents.RouterContext{AgentName: "ops-agent"}

// 入口路由
ctx := agents.RouterContext{EntryPoint: "diagnose"}

// 默认路由（EntryPoint 无匹配时）
ctx := agents.RouterContext{}
```

## 热加载

Manager 支持配置文件的热加载：

- 新建配置文件：自动加载
- 修改配置文件：自动重新加载
- 删除配置文件：自动卸载

可通过 `AutoReload` 配置启用/禁用热加载。

## 环境变量

- `AGENTS_DIR`: 指定 Agent 配置目录

## 错误处理

常见错误类型：

- `ErrAgentNotFound`: Agent 不存在
- `ErrInvalidConfig`: 配置无效
- `ErrInvalidName`: name 格式错误
- `ErrConfigNotFound`: 配置文件不存在

## 测试

```bash
go test ./internal/pkg/agents/... -v
```

## 目录结构

```
backend/internal/pkg/agents/
├── agent.go          # Agent 结构定义
├── registry.go       # Registry 接口与实现
├── parser.go         # 配置解析器
├── loader.go         # Agent 加载器
├── router.go         # Agent 路由器
├── manager.go        # 管理器（整合）
├── watcher.go        # 文件监听
├── errors.go         # 错误定义
├── types.go          # 公共类型
└── *_test.go         # 测试文件

agents/               # Agent 配置目录
├── diagnose-agent.yaml
├── ops-agent.yaml
└── default-agent.yaml
```
