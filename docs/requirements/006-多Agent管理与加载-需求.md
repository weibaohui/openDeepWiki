# 006-多Agent管理与加载-需求.md

## 1. 背景（Why）

随着系统逐步引入 **Skills（可执行能力）**、**MCP（上下文获取能力）**，仅通过向大模型直接暴露 Skills 已无法满足以下需求：

- 不同业务场景下行为模式差异巨大（诊断 / 运维 / 发布）
- 同一 Skill 在不同语境下风险和使用策略不同
- System Prompt、上下文策略、技能集合不断膨胀
- 缺乏对大模型行为的稳定性和可控性约束

因此，需要在系统中引入 **Agent（智能体）** 概念，对大模型的使用进行"场景化、角色化、策略化"的封装。

---

## 2. 目标（What，必须可验证）

- [ ] 实现 Agent 抽象层，作为会话级执行单元
- [ ] 支持多 Agent 并存，按场景选择使用
- [ ] 支持 Agent 通过 YAML 配置定义
- [ ] 支持 Agent 运行时热加载、新增、更新、删除
- [ ] 实现 Agent Manager 统一管理 Agent 的注册、卸载、查询
- [ ] 实现 Agent Router 根据请求上下文选择合适 Agent
- [ ] 支持 Agent 定义 system prompt、MCP policy、Skill policy、Runtime policy

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不实现复杂 Agent 间协作
- [ ] 不实现自学习 / 记忆型 Agent
- [ ] 不引入多模型切换机制
- [ ] 不实现 Agent 的版本管理
- [ ] 不涉及前端 UI 实现

---

## 4. 核心概念定义

### 4.1 Agent

Agent 是一个 **会话级执行单元**，其本质不是能力本身，而是：

> 对大模型的一次"角色 + 能力边界 + 行为策略"的封装。

Agent 决定：

- 我是谁（角色定位）
- 我如何思考和行动（system prompt）
- 我能看到什么上下文（MCP policy）
- 我能做哪些动作（Skill policy）
- 我在运行时受哪些限制（Runtime policy）

### 4.2 Agent Manager

Agent Manager 是系统中 Agent 的统一管理组件，负责：

- Agent 的加载 / 卸载
- Agent 的注册与查询
- Agent 定义的校验

### 4.3 Agent Router

Agent Router 负责根据请求上下文选择合适的 Agent，例如：

- 用户入口（诊断页 / 运维页）
- 显式指定 Agent
- 基于规则的初步判断

---

## 5. 功能需求清单（Checklist）

### 5.1 Agent 配置定义

- [ ] 支持 YAML 格式定义 Agent
- [ ] 基本字段：name（唯一标识）、version、description
- [ ] System Prompt 字段：定义角色和行为约束
- [ ] MCP Policy 字段：定义允许的 MCP 及调用限制
- [ ] Skill Policy 字段：定义允许的 Skills 和禁止的 Skills
- [ ] Runtime Policy 字段：定义风险等级、最大步骤数、确认要求

### 5.2 Agent 配置校验

- [ ] 校验 name 格式（小写字母、数字、连字符，最多64字符）
- [ ] 校验 version 格式（语义化版本）
- [ ] 校验 description 不为空
- [ ] 校验 MCP Policy 中引用的 MCP 是否存在于系统
- [ ] 校验 Skill Policy 中引用的 Skills 是否存在于系统
- [ ] 校验 Runtime Policy 的 riskLevel 枚举值（read / write / admin）

### 5.3 Agent 加载机制

- [ ] 默认目录：`./agents`（与可执行文件同级）
- [ ] 支持环境变量 `AGENTS_DIR` 指定目录
- [ ] 支持配置文件指定 `agents.dir`
- [ ] 系统启动时扫描并加载所有 Agent 配置
- [ ] 监听目录变化，热加载/更新/删除 Agent
- [ ] 支持手动刷新 API
- [ ] 加载失败时记录日志，不影响其他 Agent 加载

### 5.4 Agent Manager

- [ ] `Register(agent)`: 注册 Agent
- [ ] `Unregister(name)`: 注销 Agent
- [ ] `Get(name)`: 获取指定 Agent
- [ ] `List()`: 列出所有 Agents
- [ ] `Reload(name)`: 重新加载指定 Agent
- [ ] 校验 Agent 定义有效性

### 5.5 Agent Router

- [ ] `Route(context)`: 根据上下文选择合适的 Agent
- [ ] 支持按 Agent name 显式指定
- [ ] 支持按用户入口路由（如诊断页 → diagnose-agent）
- [ ] 支持按任务类型路由
- [ ] 提供默认 Agent 兜底

### 5.6 LLM 会话集成

- [ ] Agent 作为会话所有者，提供 system prompt
- [ ] 根据 Agent 的 Skill Policy 构造可用 tools
- [ ] 根据 Agent 的 MCP Policy 控制 MCP 调用
- [ ] 根据 Runtime Policy 控制会话执行（最大步骤、确认要求）
- [ ] 记录使用的 Agent（用于调试和审计）

---

## 6. 数据结构

### 6.1 Agent 结构

```go
// Agent Agent定义
type Agent struct {
    // 元数据
    Name           string            `yaml:"name" json:"name"`
    Version        string            `yaml:"version" json:"version"`
    Description    string            `yaml:"description" json:"description"`
    
    // System Prompt
    SystemPrompt   string            `yaml:"systemPrompt" json:"system_prompt"`
    
    // MCP Policy
    McpPolicy      McpPolicy         `yaml:"mcp" json:"mcp_policy"`
    
    // Skill Policy
    SkillPolicy    SkillPolicy       `yaml:"skills" json:"skill_policy"`
    
    // Runtime Policy
    RuntimePolicy  RuntimePolicy     `yaml:"policies" json:"runtime_policy"`
    
    // 路径信息
    Path           string            `json:"path"`           // Agent 配置文件路径
    LoadedAt       time.Time         `json:"loaded_at"`
}

// McpPolicy MCP策略
type McpPolicy struct {
    Allowed   []string `yaml:"allowed" json:"allowed"`     // 允许的 MCP 列表
    MaxCalls  int      `yaml:"maxCalls" json:"max_calls"`  // 最大调用次数
}

// SkillPolicy Skill策略
type SkillPolicy struct {
    Allow []string `yaml:"allow" json:"allow"`  // 显式允许的 Skills
    Deny  []string `yaml:"deny" json:"deny"`    // 显式禁止的 Skills
}

// RuntimePolicy 运行时策略
type RuntimePolicy struct {
    RiskLevel           string `yaml:"riskLevel" json:"risk_level"`                     // 风险等级：read / write / admin
    MaxSteps            int    `yaml:"maxSteps" json:"max_steps"`                       // 最大执行步骤数
    RequireConfirmation bool   `yaml:"requireConfirmation" json:"require_confirmation"` // 是否需要确认
}
```

### 6.2 Agent 加载结果

```go
// AgentLoadResult 加载结果
type AgentLoadResult struct {
    Agent   *Agent
    Error   error
    Action  string // "created", "updated", "deleted", "unchanged"
}
```

### 6.3 Router 上下文

```go
// RouterContext 路由上下文
type RouterContext struct {
    AgentName   string            // 显式指定的 Agent name
    EntryPoint  string            // 用户入口（如 "diagnose", "ops"）
    TaskType    string            // 任务类型
    Metadata    map[string]string // 附加元数据
}
```

---

## 7. Agent 配置示例

### 7.1 诊断 Agent（diagnose-agent.yaml）

```yaml
name: diagnose-agent
version: v1
description: Kubernetes 服务诊断 Agent

systemPrompt: |
  你是一个谨慎的系统诊断专家。
  你的目标是分析问题根因，而不是立即修改系统。
  在没有充分证据前，禁止执行任何写操作。
  
  你的工作流程：
  1. 收集相关日志和指标
  2. 分析异常模式
  3. 定位根因
  4. 提供修复建议（但不自动执行）

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
    - k8s-status-check
  deny:
    - restart_pod
    - scale_deployment
    - delete_resource

policies:
  riskLevel: read
  maxSteps: 6
  requireConfirmation: false
```

### 7.2 运维 Agent（ops-agent.yaml）

```yaml
name: ops-agent
version: v1
description: Kubernetes 运维操作 Agent

systemPrompt: |
  你是一个经验丰富的 Kubernetes 运维工程师。
  你可以执行常规的运维操作，但需要谨慎对待高风险操作。
  
  在执行以下操作前必须获得用户确认：
  - 删除资源
  - 重启 Pod
  - 修改配置

mcp:
  allowed:
    - cluster_state
    - pod_logs
    - metrics
    - events
  maxCalls: 10

skills:
  allow:
    - search_logs
    - analyze_logs
    - k8s-status-check
    - restart_pod
    - scale_deployment
  deny:
    - delete_resource

policies:
  riskLevel: write
  maxSteps: 10
  requireConfirmation: true
```

---

## 8. 接口设计

### 8.1 Agent Parser

```go
// Parser Agent 配置解析器
type Parser interface {
    // Parse 解析 Agent 配置文件
    Parse(configPath string) (*Agent, error)
    
    // Validate 校验 Agent 配置有效性
    Validate(agent *Agent) error
}
```

### 8.2 Agent Loader

```go
// Loader Agent 加载器
type Loader interface {
    // LoadFromDir 从目录加载所有 Agents
    LoadFromDir(dir string) ([]*AgentLoadResult, error)
    
    // LoadFromPath 加载单个 Agent
    LoadFromPath(path string) (*Agent, error)
    
    // Reload 重新加载 Agent
    Reload(name string) (*Agent, error)
    
    // Unload 卸载 Agent
    Unload(name string) error
}
```

### 8.3 Agent Manager

```go
// Manager Agent 管理器
type Manager interface {
    // Register 注册 Agent
    Register(agent *Agent) error
    
    // Unregister 注销 Agent
    Unregister(name string) error
    
    // Get 获取指定 Agent
    Get(name string) (*Agent, error)
    
    // List 列出所有 Agents
    List() []*Agent
    
    // Exists 检查 Agent 是否存在
    Exists(name string) bool
}
```

### 8.4 Agent Router

```go
// Router Agent 路由器
type Router interface {
    // Route 根据上下文选择 Agent
    Route(ctx RouterContext) (*Agent, error)
    
    // SetDefault 设置默认 Agent
    SetDefault(agentName string) error
    
    // RegisterRule 注册路由规则
    RegisterRule(rule RouteRule) error
}

// RouteRule 路由规则
type RouteRule struct {
    Priority    int               // 优先级，数字越小优先级越高
    Condition   func(RouterContext) bool
    TargetAgent string
}
```

---

## 9. 使用场景

### 场景 1：诊断场景

```
用户访问诊断页面
    ↓
Agent Router 根据 EntryPoint="diagnose" 选择 diagnose-agent
    ↓
Agent Manager 加载 diagnose-agent 定义
    ↓
系统根据 MCP Policy 获取 cluster_state, pod_logs
    ↓
构造 system prompt（诊断专家角色）+ 可用 tools（search_logs, analyze_logs）
    ↓
调用 LLM，限制 maxSteps=6，riskLevel=read
    ↓
禁止执行 restart_pod 等写操作
```

### 场景 2：运维场景

```
用户访问运维页面
    ↓
Agent Router 根据 EntryPoint="ops" 选择 ops-agent
    ↓
Agent Manager 加载 ops-agent 定义
    ↓
系统根据 MCP Policy 获取 cluster_state, pod_logs, metrics, events
    ↓
构造 system prompt（运维工程师角色）+ 可用 tools（含 restart_pod, scale_deployment）
    ↓
调用 LLM，限制 maxSteps=10，riskLevel=write
    ↓
执行高风险操作前需用户确认（requireConfirmation=true）
```

### 场景 3：显式指定 Agent

```
用户通过 API 显式指定 agent_name="diagnose-agent"
    ↓
Agent Router 直接使用指定 Agent，忽略其他规则
    ↓
如果 Agent 不存在，返回错误
```

---

## 10. 错误处理

| 错误类型 | 说明 | 处理方式 |
|---------|------|---------|
| `ErrAgentNotFound` | Agent 不存在 | 返回错误，提示可用 Agents |
| `ErrInvalidConfig` | 配置文件无效 | 记录日志，跳过该 Agent |
| `ErrInvalidName` | name 格式错误 | 记录日志，跳过该 Agent |
| `ErrMcpNotFound` | MCP Policy 引用了不存在的 MCP | 记录警告，加载时跳过无效 MCP |
| `ErrSkillNotFound` | Skill Policy 引用了不存在的 Skill | 记录警告，加载时跳过无效 Skill |
| `ErrAgentLoadFailed` | 加载失败 | 记录日志，继续加载其他 |
| `ErrAgentDirNotFound` | Agents 目录不存在 | 创建空目录，继续启动 |

---

## 11. 配置

```yaml
# config.yaml
agents:
  dir: "./agents"              # Agents 配置目录
  auto_reload: true            # 自动热加载
  reload_interval: 5           # 检查间隔（秒）
  default_agent: "default"     # 默认 Agent

  # 路由规则
  routes:
    - entry_point: "diagnose"
      agent: "diagnose-agent"
    - entry_point: "ops"
      agent: "ops-agent"
```

环境变量：
- `AGENTS_DIR`: 指定 Agents 配置目录
- `AGENTS_AUTO_RELOAD`: 是否自动热加载
- `DEFAULT_AGENT`: 默认 Agent 名称

---

## 12. 验收标准

### 12.1 功能验收

- [ ] 如果创建 `agents/my-agent/agent.yaml`，系统应自动加载
- [ ] 如果修改 agent.yaml，系统应在 5 秒内更新
- [ ] 如果删除 Agent 配置文件，系统应自动卸载
- [ ] `List()` 应返回所有已加载 Agents
- [ ] `Get(name)` 应返回指定 Agent 完整定义
- [ ] Router 应根据 EntryPoint 返回正确的 Agent
- [ ] Router 应支持显式指定 Agent name
- [ ] 未授权 Skill 不应出现在可用 tools 中
- [ ] MCP Policy 应正确限制 MCP 调用次数
- [ ] Runtime Policy 应正确限制会话执行步骤

### 12.2 配置验收

- [ ] Agent name 必须符合规范（小写、数字、连字符）
- [ ] YAML 配置必须正确解析
- [ ] 无效的 MCP/Skill 引用应记录警告但不阻断加载
- [ ] 配置校验失败应记录详细错误信息

### 12.3 安全验收

- [ ] diagnose-agent 不应能调用 restart_pod Skill
- [ ] ops-agent 调用高风险操作时应要求确认
- [ ] riskLevel=read 的 Agent 不应执行写操作
- [ ] maxSteps 应正确限制 LLM 调用步骤

---

## 13. 交付物

- [ ] Agent 核心接口定义（parser, loader, manager, router）
- [ ] Agent Manager 实现
- [ ] Agent Parser 实现（YAML/JSON 配置解析）
- [ ] Agent Loader 实现（目录扫描、热加载）
- [ ] Agent Router 实现（基于规则和上下文路由）
- [ ] Agent 配置校验器
- [ ] 示例 Agents（diagnose-agent, ops-agent）
- [ ] 单元测试
- [ ] 使用文档

---

## 14. Agent 与 Skills / MCP 的关系

```
+------------------+
|     Skills       |  <- 原子执行能力，全局注册
|  - search_logs   |
|  - restart_pod   |
+------------------+
         ↑
         | Skill Policy 决定可用 Skills
         |
+------------------+
|      Agent       |  <- 会话级执行单元
| - system prompt  |
| - MCP policy     |
| - Skill policy   |
| - Runtime policy |
+------------------+
         ↓
         | MCP Policy 决定可用 MCP
         |
+------------------+
|       MCP        |  <- 上下文获取能力
| - cluster_state  |
| - pod_logs       |
+------------------+
```

- **Skill**：原子执行能力，全局注册
- **MCP**：系统级上下文获取能力
- **Agent**：
  - 定义哪些 Skill 可用
  - 定义哪些 MCP 可用
  - 定义大模型的行为方式

> Agent 是 Skill 与 MCP 的"使用者与约束者"。

---

## 15. 会话执行流程

```
1. 用户发起请求
   ↓
2. Agent Router 根据上下文选择 Agent
   - 检查是否有显式指定的 Agent name
   - 根据 EntryPoint 匹配路由规则
   - 使用默认 Agent 兜底
   ↓
3. Agent Manager 加载 Agent 定义
   - 校验 Agent 配置有效性
   - 检查 MCP Policy 引用的 MCP 是否存在
   - 检查 Skill Policy 引用的 Skills 是否存在
   ↓
4. Agent 根据 MCP Policy 获取上下文
   - 调用允许的 MCP 获取数据
   - 控制调用次数不超过 maxCalls
   ↓
5. 系统构造会话上下文
   - system prompt = Agent.SystemPrompt
   - tools = 根据 Skill Policy 过滤后的 Skills
   ↓
6. 调用 LLM
   ↓
7. 执行 Skill 调用
   - 校验 Skill 是否在 allow 列表
   - 校验 Skill 不在 deny 列表
   - 控制执行步骤不超过 RuntimePolicy.MaxSteps
   ↓
8. 返回结果
```

---

## 16. 约束条件

### 16.1 技术约束

- Agent 配置必须以 YAML/JSON 文件形式存放，不得硬编码
- Agent name 全局唯一，不可重复
- Agent 加载失败不应影响系统启动

### 16.2 架构约束

- Agent 不得直接操作 Skills 或 MCP 的实现
- Agent 仅通过 Policy 声明使用权限
- Agent Router 不得包含业务逻辑，仅负责路由

### 16.3 安全约束

- Skill deny 列表优先级高于 allow 列表
- 高风险操作（riskLevel=admin）必须要求确认
- MCP 调用次数必须受 maxCalls 限制

### 16.4 性能约束

- 加载 20 个 Agent 配置应在 100ms 内完成
- Agent 路由应在 5ms 内完成
- 热加载检测间隔不超过 5 秒

---

## 17. 可修改 / 不可修改项

- ❌ 不可修改：
  - Agent 配置字段定义（需保持向后兼容）
  - Agent name 唯一性约束
  - riskLevel 枚举值定义
  
- ✅ 可调整：
  - 默认热加载间隔
  - 路由规则的优先级算法
  - 配置校验的严格程度
  - 错误处理方式

---

## 18. 风险与已知不确定点

| 风险点 | 说明 | 处理方式 |
|-------|------|---------|
| Agent 配置冲突 | 同名 Agent 多次定义 | 后加载的覆盖先加载的，记录警告 |
| MCP/Skill 不存在 | Policy 引用了不存在的 MCP/Skill | 记录警告，跳过无效项，不阻断加载 |
| 热加载性能 | 频繁修改配置导致频繁重载 | 添加防抖机制，最小间隔 1 秒 |
| 路由歧义 | 多个规则匹配同一上下文 | 按优先级排序，高优先级优先 |
| Agent 滥用 | 高风险 Agent 被错误路由 | 强制要求显式指定高风险 Agent |
