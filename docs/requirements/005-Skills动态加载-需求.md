# 005-Skills动态加载-需求.md

## 1. 背景（Why）

当前系统已具备基于 MCP (Model Context Protocol) 的工具调用机制（004号需求），但所有能力（Tools）在编译期静态注册，存在以下问题：

- **发布成本高**：新能力上线需要重新编译、发布主服务
- **无法按需控制**：无法运行时启停能力（如高风险写操作）
- **扩展性受限**：不利于 AI Agent 能力的扩展与演进

因此，需要引入 **Skills 动态加载机制**，使能力模块可以在运行期被加载、卸载、启用和禁用，并可被大模型自动选择和调用。

---

## 2. 目标（What，必须可验证）

- [ ] **Skills 核心框架**：定义统一的 Skill 接口规范
- [ ] **动态加载机制**：支持运行时从配置文件加载/卸载 Skills
- [ ] **Skill Registry**：实现技能注册中心，管理技能生命周期
- [ ] **HTTP Provider**：支持通过 HTTP 调用外部 Skill 服务
- [ ] **Builtin Provider**：支持内置 Go 代码实现的 Skills
- [ ] **LLM 集成**：Skills 可自动转换为 LLM Tools 供模型调用

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不实现复杂的 Agent 规划算法
- [ ] 不强制使用 plugin / wasm 等二进制加载技术
- [ ] 不涉及前端 UI 实现
- [ ] 不实现 Skill 版本管理（v1/v2）
- [ ] 不实现多 Skill 编排（Plan → Execute）
- [ ] 不实现 RBAC 权限控制（仅预留风险等级标记）

---

## 4. 核心概念定义

### 4.1 Skill（技能）

Skill 是系统中的最小"能力单元"，代表一个可被 AI 调用的原子能力。

一个 Skill 必须具备：

| 属性 | 说明 |
|------|------|
| `name` | 唯一名称（全局唯一） |
| `description` | 描述信息，供 LLM 理解用途 |
| `parameters` | 输入参数定义（JSON Schema） |
| `execute` | 执行逻辑 |

### 4.2 Skill Provider（技能提供者）

Skill Provider 是 Skill 的加载来源：

| Provider 类型 | 说明 |
|--------------|------|
| `builtin` | 内置 Go 实现，编译到主程序中 |
| `http` | 外部 HTTP 服务，通过 HTTP 调用 |

### 4.3 Skill Registry（技能注册中心）

Skill Registry 负责：

- 维护当前已加载的 Skills
- 管理 Skills 的启用/禁用状态
- 为 LLM 暴露可用 Skills 列表

### 4.4 Skill 状态模型

```
+--------+    Enable     +---------+
| loaded | ------------> | enabled |
+--------+               +---------+
   |  |                     |  |
   |  +-- Disable ----+     |  |
   |                   |     |  |
   +---- Unregister ---+     +-- Unregister
                              |
                              v
                           +---------+
                           | removed |
                           +---------+
```

| 状态 | 说明 |
|------|------|
| `loaded` | 已加载到 Registry |
| `enabled` | 可被 LLM 调用 |
| `disabled` | 不可调用但仍保留 |

---

## 5. 功能需求清单（Checklist）

### 5.1 Skill 接口规范

所有 Skills 必须实现统一接口：

```go
// Skill 技能接口
type Skill interface {
    // Name 返回技能唯一名称
    Name() string
    
    // Description 返回技能描述
    Description() string
    
    // Parameters 返回参数 JSON Schema
    Parameters() ParameterSchema
    
    // Execute 执行技能
    // ctx: 上下文
    // args: JSON 格式的参数
    // 返回: 执行结果（JSON 可序列化）和错误
    Execute(ctx context.Context, args json.RawMessage) (interface{}, error)
}
```

设计约束：
- [ ] 参数与返回值必须可 JSON 序列化
- [ ] Execute 必须是同步接口（异步由上层编排）
- [ ] Skill 不直接依赖 LLM SDK

### 5.2 Skill Registry

#### 5.2.1 基本能力

| 方法 | 说明 |
|------|------|
| `Register(skill Skill) error` | 注册 Skill |
| `Unregister(name string) error` | 注销 Skill |
| `Enable(name string) error` | 启用 Skill |
| `Disable(name string) error` | 禁用 Skill |
| `Get(name string) (Skill, error)` | 获取 Skill |
| `List() []Skill` | 列出所有 Skills |
| `ListEnabled() []Skill` | 列出已启用的 Skills |

#### 5.2.2 线程安全

- [ ] Registry 必须线程安全，支持并发访问
- [ ] 所有操作必须加锁保护

### 5.3 Skill Provider 实现

#### 5.3.1 Builtin Provider（内置提供者）

- [ ] 支持 Go 代码直接实现 Skill 接口
- [ ] 在系统启动或运行期注册
- [ ] 用于核心、高频、低延迟能力

#### 5.3.2 HTTP Provider（外部 HTTP 服务）

- [ ] 通过 HTTP 调用外部 Skill 服务
- [ ] 支持配置 endpoint、timeout、headers
- [ ] 便于多语言、多团队扩展

HTTP 调用约定：

| 项目 | 说明 |
|------|------|
| 方法 | POST |
| 路径 | 可配置（默认 `/execute`） |
| 请求 Body | JSON（Skill 参数） |
| 响应 Body | JSON（执行结果） |
| 超时 | 可配置（默认 30s） |

### 5.4 动态加载机制

#### 5.4.1 配置目录

- [ ] 默认目录：`./skills`（与可执行文件同级目录）
- [ ] 支持环境变量 `SKILLS_DIR` 指定目录
- [ ] 支持配置文件 `config.yaml` 中指定 `skills.dir`

#### 5.4.2 Skill 描述文件

每个 Skill 由一个 YAML 文件描述：

```yaml
# skills/search_logs.yaml
name: search_logs
description: 搜索 Kubernetes Pod 日志
provider: http
endpoint: http://127.0.0.1:8081/execute
timeout: 30
headers:
  Authorization: Bearer ${TOKEN}
risk_level: read  # read / write / destructive
parameters:
  type: object
  properties:
    namespace:
      type: string
      description: Kubernetes 命名空间
    pod:
      type: string
      description: Pod 名称
    keyword:
      type: string
      description: 搜索关键词
  required:
    - namespace
    - pod
```

#### 5.4.3 文件监听与热加载

- [ ] 系统启动时加载目录下所有 Skill 配置
- [ ] 监听目录变更（文件新增/修改/删除）
- [ ] 变更后自动加载/更新/卸载 Skill
- [ ] 支持手动触发重新加载（API 或信号）

### 5.5 与 LLM 的集成

#### 5.5.1 Skill → Tool 映射

系统需将 `enabled` 状态的 Skills 转换为 LLM 可识别的 Tools：

| Skill 字段 | Tool 字段 |
|-----------|----------|
| `name` | `function.name` |
| `description` | `function.description` |
| `parameters` | `function.parameters` |

#### 5.5.2 调用流程

```
1. 系统向 LLM 发送对话 + Tools 列表（由 enabled Skills 转换）
2. LLM 返回 tool_calls
3. Skill Router 根据 name 找到对应 Skill
4. 调用 Skill.Execute(ctx, args)
5. 将结果回填给 LLM
```

### 5.6 错误处理

标准化错误类型：

| 错误类型 | 说明 | HTTP 状态码 |
|---------|------|------------|
| `ErrSkillNotFound` | Skill 不存在 | 404 |
| `ErrSkillDisabled` | Skill 已禁用 | 403 |
| `ErrInvalidParams` | 参数校验失败 | 400 |
| `ErrProviderTimeout` | Provider 超时 | 504 |
| `ErrProviderUnavailable` | Provider 不可达 | 502 |
| `ErrExecutionFailed` | 执行失败 | 500 |

### 5.7 安全控制（预留）

- [ ] Skill 风险等级标记（`read` / `write` / `destructive`）
- [ ] 预留调用白名单/黑名单机制接口
- [ ] 预留 RBAC 集成点

---

## 6. 约束条件

### 6.1 技术约束

- 必须使用 Go 语言实现，符合项目现有编码规范
- 必须兼容 OpenAI API 格式的 Function Calling 协议
- Skill 配置文件必须是有效的 YAML 格式
- 所有输入输出必须可 JSON 序列化

### 6.2 架构约束

- Skill 接口与具体业务解耦
- Registry 必须线程安全
- Provider 实现必须可扩展
- 不得修改现有的 Tool 调用流程（向后兼容）

### 6.3 性能约束

- Skill 注册/注销操作 O(1)
- Skill 列表查询 O(n)，n 为 Skill 数量
- HTTP Provider 默认超时 30s，最大 120s
- 配置文件监听检测间隔 ≤ 5s

### 6.4 安全约束

- Skill 配置文件路径必须是绝对路径或相对于工作目录
- 禁止加载工作目录外的 Skill 配置文件
- HTTP Provider 必须验证响应内容类型

---

## 7. 可修改 / 不可修改项

| 项目 | 可否修改 | 说明 |
|------|---------|------|
| Skill 接口定义 | ❌ 不可修改 | 核心契约 |
| Registry 方法签名 | ❌ 不可修改 | 公共 API |
| Skill 配置文件格式 | ✅ 可调整 | 可新增字段 |
| 风险等级枚举值 | ✅ 可调整 | 可扩展 |
| HTTP 调用约定 | ✅ 可调整 | 路径、超时等 |

---

## 8. 接口与数据约定

### 8.1 Skill 配置结构

```go
// SkillConfig Skill 配置文件结构
type SkillConfig struct {
    Name        string          `yaml:"name" json:"name"`
    Description string          `yaml:"description" json:"description"`
    Provider    string          `yaml:"provider" json:"provider"` // builtin / http
    Endpoint    string          `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
    Timeout     int             `yaml:"timeout,omitempty" json:"timeout,omitempty"`
    Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
    RiskLevel   string          `yaml:"risk_level,omitempty" json:"risk_level,omitempty"` // read / write / destructive
    Parameters  ParameterSchema `yaml:"parameters" json:"parameters"`
}
```

### 8.2 Registry 接口

```go
// Registry Skill 注册中心接口
type Registry interface {
    Register(skill Skill) error
    Unregister(name string) error
    Enable(name string) error
    Disable(name string) error
    Get(name string) (Skill, error)
    List() []Skill
    ListEnabled() []Skill
    ToTools() []llm.Tool  // 转换为 LLM Tools
}
```

### 8.3 Provider 接口

```go
// Provider Skill 提供者接口
type Provider interface {
    // Load 从配置加载 Skill
    Load(config SkillConfig) (Skill, error)
    // Type 返回 Provider 类型
    Type() string
}
```

---

## 9. 验收标准（Acceptance Criteria）

### 9.1 功能验收

- [ ] 如果创建 Skill 配置文件，系统应自动加载并注册
- [ ] 如果修改 Skill 配置文件，系统应自动更新
- [ ] 如果删除 Skill 配置文件，系统应自动注销
- [ ] 如果调用 `ListEnabled()`，应返回所有已启用的 Skills
- [ ] 如果将 Skills 转换为 Tools，应符合 OpenAI Function Calling 格式
- [ ] 如果调用 HTTP Provider Skill，应正确转发请求到配置的 endpoint

### 9.2 性能验收

- [ ] Skill 注册操作应在 10ms 内完成
- [ ] 100 个 Skill 的列表查询应在 50ms 内完成
- [ ] 配置文件变更应在 5s 内被检测并加载

### 9.3 安全验收

- [ ] 如果 Skill 配置文件位于工作目录外，应拒绝加载
- [ ] 如果 HTTP Provider 响应非 JSON，应返回错误

### 9.4 稳定性验收

- [ ] Skill 执行失败不应影响主服务稳定性
- [ ] Registry 操作并发安全，无数据竞争

---

## 10. 风险与已知不确定点

| 风险 | 影响 | 对策 |
|------|------|------|
| 配置文件格式错误导致启动失败 | 高 | 加载失败时记录错误但继续启动；提供配置校验工具 |
| HTTP Provider 服务不可用 | 中 | 实现健康检查；支持超时和重试 |
| Skill 名称冲突 | 中 | 后加载的覆盖先加载的；记录警告日志 |
| 文件监听在某些系统上不工作 | 低 | 支持手动刷新 API；提供定时轮询作为后备 |
| 动态加载导致运行时行为不一致 | 低 | 提供 Registry 状态查询 API；记录操作日志 |

---

## 11. 依赖关系

- **前置依赖**：004-MCP工具支持（已提供 Tool 定义和 LLM 集成基础）
- **后续可扩展**：Agent 规划算法、多 Skill 编排

---

## 12. 交付物

- [ ] Skills 核心接口定义（`pkg/skills/`）
- [ ] Skill Registry 实现
- [ ] HTTP Provider 实现
- [ ] 配置加载与文件监听实现
- [ ] Builtin Provider 示例
- [ ] 示例 Skill 配置（search_logs.yaml）
- [ ] 单元测试

---

## 13. 代码目录结构

```
backend/internal/pkg/skills/
├── registry.go           # Registry 接口与实现
├── skill.go              # Skill 接口定义
├── provider.go           # Provider 接口
├── config.go             # 配置结构定义
├── loader.go             # 配置文件加载与监听
├── builtin/
│   └── provider.go       # Builtin Provider 实现
└── http/
    ├── provider.go       # HTTP Provider 实现
    └── client.go         # HTTP 客户端

# 示例 Skill 配置
skills/
└── search_logs.yaml      # 示例 HTTP Skill
```

---

## 14. 后续优化方向

- [ ] Skill 版本管理（v1 / v2）
- [ ] 多 Skill 编排（Plan → Execute）
- [ ] Skill 调用观测（Tracing / Metrics）
- [ ] WASM / Plugin Provider
- [ ] Skill 调用历史记录和审计
- [ ] RBAC 权限控制集成
