# Skills 动态加载框架

## 概述

Skills 是一个动态加载的能力框架，允许在运行时加载、卸载、启用和禁用能力模块，并可被大模型自动选择和调用。

## 核心概念

- **Skill**: 最小能力单元，包含名称、描述、参数定义和执行逻辑
- **Provider**: Skill 的提供者，支持 `builtin`（内置）和 `http`（外部 HTTP 服务）
- **Registry**: 技能注册中心，管理技能的生命周期
- **Manager**: 管理器，负责初始化、配置加载和文件监听

## 快速开始

### 1. 初始化 Manager

```go
import "github.com/weibh/openDeepWiki/backend/internal/pkg/skills"

// 使用默认目录（./skills 或 SKILLS_DIR 环境变量）
manager, err := skills.NewManager("")
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()

// 或指定目录
manager, err := skills.NewManager("/path/to/skills")
```

### 2. 注册内置 Skill

```go
import (
    "context"
    "encoding/json"
    "github.com/weibh/openDeepWiki/backend/internal/pkg/skills/builtin"
)

// 方式1：注册创建器
manager.RegisterBuiltinCreator("my_skill", func(config skills.SkillConfig) (skills.Skill, error) {
    return builtin.NewBuiltinSkill(
        "my_skill",
        "我的技能描述",
        skills.ParameterSchema{
            Type: "object",
            Properties: map[string]skills.Property{
                "input": {Type: "string", Description: "输入参数"},
            },
            Required: []string{"input"},
        },
        func(ctx context.Context, args json.RawMessage) (interface{}, error) {
            var params struct {
                Input string `json:"input"`
            }
            if err := json.Unmarshal(args, &params); err != nil {
                return nil, err
            }
            return map[string]string{"result": params.Input}, nil
        },
    ), nil
})

// 方式2：直接注册 Skill 实例
skill := builtin.NewBuiltinSkill(...)
manager.RegisterBuiltinSkill(skill)
```

### 3. 在 LLM Client 中使用

```go
import "github.com/weibh/openDeepWiki/backend/internal/pkg/llm"

// 创建 Skill 执行器
executor := skills.NewExecutor(manager.Registry)

// 获取 Tools（从 enabled Skills 转换）
tools := manager.Registry.ToTools()

// 执行对话
messages := []llm.ChatMessage{
    {Role: "system", Content: "你是一个助手..."},
    {Role: "user", Content: "请使用技能..."},
}

response, err := client.ChatWithToolExecution(ctx, messages, tools, executor)
```

## 配置文件

Skill 配置文件放在 `skills/` 目录下，支持 YAML 和 JSON 格式。

### HTTP Skill 示例

```yaml
# skills/my_api.yaml
name: my_api
description: 调用外部 API 服务
provider: http
endpoint: http://localhost:8080/api/execute
timeout: 30
headers:
  Authorization: Bearer ${API_TOKEN}
  Content-Type: application/json
risk_level: read
parameters:
  type: object
  properties:
    query:
      type: string
      description: 查询参数
    limit:
      type: integer
      description: 返回数量限制
      default: 10
  required:
    - query
```

### 配置字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Skill 唯一名称 |
| description | string | 是 | Skill 描述 |
| provider | string | 是 | Provider 类型：`builtin` 或 `http` |
| endpoint | string | HTTP 必填 | HTTP 服务端点 |
| timeout | int | 否 | 超时时间（秒），默认 30 |
| headers | map | 否 | HTTP 请求头 |
| risk_level | string | 否 | 风险等级：`read`/`write`/`destructive` |
| parameters | object | 是 | JSON Schema 参数定义 |

## 动态加载

Manager 会自动监听 `skills/` 目录的变化：

- **新增文件**: 自动加载并注册 Skill
- **修改文件**: 自动更新 Skill
- **删除文件**: 自动注销 Skill

监听间隔为 5 秒。

## 目录结构

```
skills/
├── skill.go              # Skill 接口定义
├── registry.go           # Registry 实现
├── provider.go           # Provider 接口
├── config.go             # 配置结构
├── loader.go             # 配置加载
├── watcher.go            # 文件监听
├── manager.go            # 管理器
├── executor.go           # LLM 执行器
├── errors.go             # 错误定义
├── builtin/
│   └── provider.go       # Builtin Provider
└── http/
    ├── provider.go       # HTTP Provider
    ├── skill.go          # HTTP Skill
    └── client.go         # HTTP 客户端
```

## 扩展 Provider

实现自定义 Provider：

```go
type MyProvider struct{}

func (p *MyProvider) Type() string {
    return "my_provider"
}

func (p *MyProvider) Create(config skills.SkillConfig) (skills.Skill, error) {
    // 创建并返回 Skill 实例
    return &MySkill{config: config}, nil
}

// 注册 Provider
manager.providers.Register(&MyProvider{})
```
