# 013-ADKAgents管理模块-需求.md

## 1. 背景（Why）

当前系统使用 Eino ADK（Agent Development Kit）实现了文档生成工作流，Agent 定义位于 `backend/internal/service/einodoc/adk/agents.go` 中，采用代码硬编码方式创建：

- `CreateRepoInitializerAgent()` - 仓库初始化 Agent
- `CreateArchitectAgent()` - 架构师 Agent  
- `CreateExplorerAgent()` - 探索者 Agent
- `CreateWriterAgent()` - 作者 Agent
- `CreateEditorAgent()` - 编辑 Agent
- `CreateSequentialAgent()` - 顺序执行组合 Agent

这种方式存在以下问题：

1. **缺乏灵活性**：Agent 配置硬编码在 Go 代码中，修改需要重新编译
2. **无法热更新**：无法在不重启服务的情况下更新 Agent 配置
3. **配置分散**：SystemPrompt、ToolsConfig、MaxIterations 等配置散落在代码各处
4. **无法动态扩展**：新增 Agent 需要修改代码并重新部署
5. **与既有 agents 模块重复**：`backend/internal/pkg/agents/` 模块已实现基于 YAML 的 Agent 管理，但未与 ADK 集成

因此，需要构建新的 `adkagents` 模块，实现基于 YAML 配置的 ADK Agent 管理，支持热加载、动态扩展，并统一管理 ADK Agent 的生命周期。

---

## 2. 目标（What，必须可验证）

- [ ] 新建 `backend/internal/pkg/adkagents/` 模块，实现 ADK Agent 的 YAML 配置化管理
- [ ] 支持从 YAML 文件加载 ADK Agent 配置，适配 Eino ADK 的 `ChatModelAgentConfig`
- [ ] 实现 `NewManager(config *Config).GetAgent(name)` 接口，替代原有的 `CreateXxxAgent()` 函数
- [ ] 将 `backend/internal/service/einodoc/adk/agents.go` 中的 Agent 定义迁移到 `agents/` 目录下的 YAML 文件
- [ ] 一个 Agent 对应一个 YAML 配置文件（如 `agents/repo-initializer.yaml`）
- [ ] 支持 Agent 运行时热加载、更新、删除
- [ ] 保持 `CreateSequentialAgent()` 等组合型 Agent 的既有逻辑，不由 adkagents 管理
- [ ] 重构 `backend/internal/pkg/agents/` 模块，移除未启用的旧代码，不保留兼容性

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不实现 Agent 间协作编排（如 SequentialAgent、LoopAgent 等组合逻辑）
- [ ] 不实现 ADK Workflow 的 YAML 配置化（仅管理单个 Agent）
- [ ] 不实现 Agent 的版本管理
- [ ] 不引入多模型切换机制（每个 Agent 固定使用配置中指定的模型）
- [ ] 不涉及前端 UI 实现

---

## 4. 核心概念定义

### 4.1 ADK Agent

ADK Agent 是基于 Eino ADK 的 `ChatModelAgent`，其配置包括：

- **Name**：Agent 唯一标识
- **Description**：Agent 描述
- **Instruction**：System Prompt，定义角色和行为约束
- **Model**：使用的 LLM 模型
- **ToolsConfig**：工具配置（允许使用的 Tools 列表）
- **MaxIterations**：最大迭代次数
- **Exit**：退出条件配置（可选）

### 4.2 ADK Agent Manager

ADK Agent Manager 是 ADK Agent 的统一管理组件，负责：

- Agent 配置文件的加载 / 卸载
- Agent 运行时实例的创建与缓存
- Agent 定义的热加载与更新
- 提供 `GetAgent(name)` 接口获取 ADK Agent 实例

### 4.3 基础 Agent vs 组合 Agent

| 类型 | 说明 | 管理方 |
|------|------|--------|
| 基础 Agent | 单一职责的 ChatModelAgent，如 RepoInitializer、Architect | adkagents.Manager |
| 组合 Agent | 由多个基础 Agent 组合而成，如 SequentialAgent | 既有逻辑（einodoc/adk/agents.go） |

> **重要原则**：adkagents.Manager 只管理基础 Agent 的装配，组合型 Agent 的创建逻辑保持既有实现。

---

## 5. 功能需求清单（Checklist）

### 5.1 ADK Agent YAML 配置定义

- [ ] 支持 YAML 格式定义 ADK Agent 配置
- [ ] 基本字段：`name`（唯一标识）、`description`（描述）
- [ ] Instruction 字段：定义 System Prompt
- [ ] Model 字段：指定使用的 LLM 模型（默认/指定模型名）
- [ ] Tools 字段：定义允许使用的 Tools 列表（引用 `tools/` 目录下的工具定义）
- [ ] MaxIterations 字段：最大迭代次数
- [ ] Exit 字段：退出条件配置（可选）

### 5.2 Agent 配置校验

- [ ] 校验 `name` 格式（小写字母、数字、连字符，最多64字符）
- [ ] 校验 `name` 全局唯一性
- [ ] 校验 `description` 不为空
- [ ] 校验 `instruction` 不为空
- [ ] 校验 `maxIterations` 为正整数
- [ ] 校验失败时记录详细错误日志，跳过该 Agent

### 5.3 Agent 加载机制

- [ ] 默认目录：`./agents`（与可执行文件同级）
- [ ] 支持环境变量 `AGENTS_DIR` 指定目录
- [ ] 支持配置文件指定 `agents.dir`
- [ ] 系统启动时扫描并加载所有 Agent 配置
- [ ] 监听目录变化，热加载/更新/删除 Agent
- [ ] 支持手动刷新 API
- [ ] 加载失败时记录日志，不影响其他 Agent 加载

### 5.4 ADK Agent Manager

- [ ] `NewManager(config *Config)`: 创建 Manager 实例
- [ ] `GetAgent(name) (adk.Agent, error)`: 获取指定名称的 ADK Agent 实例
- [ ] `Register(agentDef *AgentDefinition)`: 注册 Agent 定义
- [ ] `Unregister(name)`: 注销 Agent
- [ ] `List() []*AgentDefinition`: 列出所有 Agent 定义
- [ ] `Reload(name)`: 重新加载指定 Agent
- [ ] 缓存已创建的 ADK Agent 实例，避免重复创建
- [ ] Agent 配置更新时，清除缓存，下次 `GetAgent` 时重新创建

### 5.5 工具解析与绑定

- [ ] 解析 YAML 中的 `tools` 列表，映射到实际的 Tool 实例
- [ ] 支持引用 `tools/` 目录下注册的工具
- [ ] 工具不存在时记录警告，跳过该工具，不阻断 Agent 加载
- [ ] 构造 `compose.ToolsNodeConfig` 传递给 ADK

### 5.6 模型解析与绑定

- [ ] 支持配置模型名称或模型别名
- [ ] 从系统的 LLM Client 获取对应的 ChatModel 实例
- [ ] 未指定模型时使用默认模型

---

## 6. 数据结构

### 6.1 ADK Agent 定义结构（YAML 配置）

```go
// AgentDefinition ADK Agent 定义（从 YAML 加载）
type AgentDefinition struct {
    // 元数据
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description" json:"description"`
    
    // LLM 配置
    Model       string `yaml:"model" json:"model"` // 模型名称或别名，空则使用默认
    
    // Agent 行为配置
    Instruction   string   `yaml:"instruction" json:"instruction"`     // System Prompt
    Tools         []string `yaml:"tools" json:"tools"`                 // 工具名称列表
    MaxIterations int      `yaml:"maxIterations" json:"max_iterations"` // 最大迭代次数
    
    // 可选配置
    Exit ExitConfig `yaml:"exit,omitempty" json:"exit,omitempty"` // 退出条件
    
    // 路径信息（运行时填充）
    Path     string    `json:"path"`      // 配置文件路径
    LoadedAt time.Time `json:"loaded_at"` // 加载时间
}

// ExitConfig 退出条件配置
type ExitConfig struct {
    Type string `yaml:"type" json:"type"` // 退出类型，如 "tool_call"
}
```

### 6.2 YAML 配置示例

```yaml
# agents/repo-initializer.yaml
name: RepoInitializer
description: 仓库初始化专员 - 负责对代码仓库进行初步分析，获取目录结构

model: ""  # 使用默认模型

instruction: |
  你的任务是：
  1. 使用 list_dir 工具读取仓库的目录结构
  2. 返回仓库的完整信息，包括：
     - 仓库 URL
     - 本地路径
     - 目录结构概要

  请确保：
  - 获取完整的目录结构
  - 返回的信息准确完整

tools:
  - list_dir

maxIterations: 10
```

```yaml
# agents/architect.yaml
name: Architect
description: 文档架构师 - 负责分析仓库类型并生成文档大纲

instruction: |
  你的任务是分析仓库并生成文档大纲：
  1. 分析仓库的目录结构
  2. 识别仓库类型（go/java/python/frontend/mixed）
  3. 识别主要技术栈
  4. 生成 2-3 个章节的文档大纲

  输出格式必须是 JSON：
  {
    "repo_type": "go",
    "tech_stack": ["Go", "Gin", "GORM"],
    "summary": "项目简介",
    "chapters": [
      {
        "title": "章节标题",
        "sections": [
          {"title": "小节标题", "hints": ["提示1", "提示2"]}
        ]
      }
    ]
  }

  请确保输出格式正确，可以被 JSON 解析。

tools:
  - search_files
  - list_dir
  - read_file

maxIterations: 5
```

```yaml
# agents/editor.yaml
name: Editor
description: 文档编辑 - 负责组装和优化最终文档

instruction: |
  你是文档编辑 Editor。
  你的职责是：
  1. 组装所有章节内容形成完整文档
  2. 优化文档结构和格式
  3. 添加文档头部信息（标题、仓库信息、技术栈）
  4. 确保 Markdown 格式规范
  5. 添加目录和导航链接

  输出要求：
  - 完整的 Markdown 文档
  - 格式规范
  - 结构清晰
  - 可直接发布

# Editor 不需要 tools
tools: []

maxIterations: 5

exit:
  type: tool_call
```

### 6.3 Manager 配置

```go
// Config ADK Agent Manager 配置
type Config struct {
    Dir            string        // Agent 配置文件目录
    AutoReload     bool          // 是否启用热加载
    ReloadInterval time.Duration // 热加载检查间隔
    
    // 依赖注入
    ModelProvider  ModelProvider // 模型提供者，用于获取 ChatModel 实例
    ToolProvider   ToolProvider  // 工具提供者，用于获取 Tool 实例
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
    return &Config{
        Dir:            "./agents",
        AutoReload:     true,
        ReloadInterval: 5 * time.Second,
    }
}

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

### 6.4 Manager 结构

```go
// Manager ADK Agent 管理器
type Manager struct {
    config     *Config
    registry   *Registry                    // Agent 定义注册表
    cache      map[string]adk.Agent         // ADK Agent 实例缓存
    cacheMutex sync.RWMutex
    
    loader     *Loader      // 配置加载器
    parser     *Parser      // 配置解析器
    watcher    *FileWatcher // 文件监听器
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error)

// GetAgent 获取指定名称的 ADK Agent 实例
// 如果缓存中不存在，根据 AgentDefinition 创建并缓存
func (m *Manager) GetAgent(name string) (adk.Agent, error)

// List 列出所有 Agent 定义
func (m *Manager) List() []*AgentDefinition

// Reload 重新加载指定 Agent
func (m *Manager) Reload(name string) error

// Stop 停止 Manager，关闭文件监听
func (m *Manager) Stop()
```

---

## 7. 接口设计

### 7.1 Agent Parser

```go
// Parser ADK Agent 配置解析器
type Parser struct {
    maxNameLen        int
    maxDescriptionLen int
}

// NewParser 创建解析器
func NewParser() *Parser

// Parse 解析 Agent 配置文件
func (p *Parser) Parse(configPath string) (*AgentDefinition, error)

// Validate 校验 Agent 定义
func (p *Parser) Validate(def *AgentDefinition, toolProvider ToolProvider) error
```

### 7.2 Agent Loader

```go
// Loader ADK Agent 配置加载器
type Loader struct {
    parser   *Parser
    registry *Registry
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry *Registry) *Loader

// LoadFromDir 从目录加载所有 Agent 配置
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error)

// LoadFromPath 加载单个 Agent 配置
func (l *Loader) LoadFromPath(path string) (*AgentDefinition, error)

// Reload 重新加载指定 Agent
func (l *Loader) Reload(name string) (*AgentDefinition, error)

// Unload 卸载 Agent
func (l *Loader) Unload(name string) error
```

### 7.3 Agent Registry

```go
// Registry Agent 定义注册表
type Registry struct {
    agents map[string]*AgentDefinition
    mutex  sync.RWMutex
}

// NewRegistry 创建注册表
func NewRegistry() *Registry

// Register 注册 Agent 定义
func (r *Registry) Register(def *AgentDefinition) error

// Unregister 注销 Agent
func (r *Registry) Unregister(name string) error

// Get 获取 Agent 定义
func (r *Registry) Get(name string) (*AgentDefinition, error)

// List 列出所有 Agent 定义
func (r *Registry) List() []*AgentDefinition

// Exists 检查 Agent 是否存在
func (r *Registry) Exists(name string) bool
```

---

## 8. 迁移方案

### 8.1 原代码迁移对照表

| 原函数 | 新方式 | YAML 文件 |
|--------|--------|-----------|
| `CreateRepoInitializerAgent()` | `manager.GetAgent("RepoInitializer")` | `agents/repo-initializer.yaml` |
| `CreateArchitectAgent()` | `manager.GetAgent("Architect")` | `agents/architect.yaml` |
| `CreateExplorerAgent()` | `manager.GetAgent("Explorer")` | `agents/explorer.yaml` |
| `CreateWriterAgent()` | `manager.GetAgent("Writer")` | `agents/writer.yaml` |
| `CreateEditorAgent()` | `manager.GetAgent("Editor")` | `agents/editor.yaml` |

### 8.2 保持不变的代码

```go
// CreateSequentialAgent 保持既有实现，不由 adkagents.Manager 管理
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
    ctx := context.Background()

    // 通过 manager.GetAgent() 获取基础 Agent
    initializer, err := f.manager.GetAgent(AgentRepoInitializer)
    if err != nil {
        return nil, err
    }

    architect, err := f.manager.GetAgent(AgentArchitect)
    if err != nil {
        return nil, err
    }

    explorer, err := f.manager.GetAgent(AgentExplorer)
    if err != nil {
        return nil, err
    }

    writer, err := f.manager.GetAgent(AgentWriter)
    if err != nil {
        return nil, err
    }

    editor, err := f.manager.GetAgent(AgentEditor)
    if err != nil {
        return nil, err
    }

    // 创建 SequentialAgent（组合逻辑保持不变）
    config := &adk.SequentialAgentConfig{
        Name:        "RepoDocSequentialAgent",
        Description: "仓库文档生成顺序执行 Agent",
        SubAgents: []adk.Agent{
            initializer,
            architect,
            explorer,
            writer,
            editor,
        },
    }

    return adk.NewSequentialAgent(ctx, config)
}
```

### 8.3 重构后的 AgentFactory

```go
// AgentFactory 负责创建各种 Agent
type AgentFactory struct {
    manager  *adkagents.Manager  // 使用 adkagents.Manager 替代 chatModel/basePath
    basePath string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(manager *adkagents.Manager, basePath string) *AgentFactory {
    return &AgentFactory{
        manager:  manager,
        basePath: basePath,
    }
}

// CreateSequentialAgent 创建顺序执行的 SequentialAgent（保持既有逻辑）
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
    // ... 如上所示
}
```

---

## 9. 文件结构

```
backend/internal/pkg/adkagents/
├── agent.go           # AgentDefinition 结构定义
├── manager.go         # Manager 实现
├── parser.go          # YAML 配置解析器
├── loader.go          # 配置加载器
├── registry.go        # Agent 定义注册表
├── watcher.go         # 文件变更监听
├── errors.go          # 错误定义
├── types.go           # 类型定义
└── README.md          # 使用文档

agents/                # Agent 配置文件目录（已存在，需调整格式）
├── repo-initializer.yaml   # RepoInitializer Agent 配置
├── architect.yaml          # Architect Agent 配置
├── explorer.yaml           # Explorer Agent 配置
├── writer.yaml             # Writer Agent 配置
├── editor.yaml             # Editor Agent 配置
└── ...                     # 其他 Agent 配置
```

---

## 10. 配置

```yaml
# config.yaml
agents:
  dir: "./agents"              # Agent 配置目录
  auto_reload: true            # 自动热加载
  reload_interval: 5           # 检查间隔（秒）
```

环境变量：
- `AGENTS_DIR`: 指定 Agent 配置目录

---

## 11. 错误处理

| 错误类型 | 说明 | 处理方式 |
|---------|------|---------|
| `ErrAgentNotFound` | Agent 不存在 | 返回错误，提示可用 Agents |
| `ErrInvalidConfig` | 配置文件无效 | 记录日志，跳过该 Agent |
| `ErrInvalidName` | name 格式错误 | 记录日志，跳过该 Agent |
| `ErrToolNotFound` | Tools 中引用了不存在的工具 | 记录警告，跳过该工具 |
| `ErrModelNotFound` | 指定的模型不存在 | 使用默认模型 |
| `ErrAgentLoadFailed` | 加载失败 | 记录日志，继续加载其他 |
| `ErrAgentDirNotFound` | Agent 目录不存在 | 创建空目录，继续启动 |

---

## 12. 验收标准

### 12.1 功能验收

- [ ] 如果创建 `agents/my-agent.yaml`，系统应自动加载
- [ ] 如果修改 agent.yaml，系统应在 5 秒内更新
- [ ] 如果删除 Agent 配置文件，系统应自动卸载
- [ ] `manager.GetAgent(name)` 应返回有效的 `adk.Agent` 实例
- [ ] `manager.List()` 应返回所有已加载 Agent 定义
- [ ] Agent 配置中的 tools 应正确绑定到 ADK Agent
- [ ] Agent 配置中的 instruction 应作为 System Prompt
- [ ] Agent 配置更新后，下次 `GetAgent` 应返回新的实例
- [ ] 加载失败时，不影响其他 Agent 的加载和使用

### 12.2 配置验收

- [ ] Agent name 必须符合规范（小写、数字、连字符）
- [ ] YAML 配置必须正确解析
- [ ] 无效的工具引用应记录警告但不阻断加载
- [ ] 配置校验失败应记录详细错误信息

### 12.3 迁移验收

- [ ] 原有的 `CreateRepoInitializerAgent()` 等函数可替换为 `manager.GetAgent()`
- [ ] `CreateSequentialAgent()` 功能保持正常，通过 `GetAgent` 获取子 Agent
- [ ] 文档生成工作流功能不受影响

---

## 13. 交付物

- [ ] ADK Agent 核心接口定义（parser, loader, registry, manager）
- [ ] ADK Agent Manager 实现
- [ ] YAML 配置解析器实现
- [ ] 配置加载器实现（目录扫描、热加载）
- [ ] Agent 定义注册表实现
- [ ] 文件变更监听器实现
- [ ] 5 个基础 Agent 的 YAML 配置文件（RepoInitializer, Architect, Explorer, Writer, Editor）
- [ ] 单元测试
- [ ] 使用文档
- [ ] 迁移后的 `einodoc/adk/agents.go`（简化版，仅保留组合逻辑）

---

## 14. 与既有模块的关系

### 14.1 与原有 agents 模块的关系

```
backend/internal/pkg/
├── agents/              # 原有模块（未启用，本次重构后移除或归档）
│   ├── agent.go
│   ├── manager.go
│   └── ...
│
└── adkagents/           # 新模块（本次新建）
    ├── agent.go
    ├── manager.go
    └── ...
```

> 说明：原有 `agents` 模块未在系统中启用，本次重构后直接移除，不保留兼容性。

### 14.2 与 tools 模块的关系

```
+---------------------+
|   tools/            |  <- Tool 定义与注册
|  - list_dir         |
|  - read_file        |
|  - search_files     |
+---------------------+
          ↑
          | ToolProvider 获取工具实例
          |
+---------------------+
|   adkagents/        |  <- ADK Agent 管理
|  - Manager          |
|  - AgentDefinition  |
+---------------------+
          ↓
          | 构造 ToolsNodeConfig
          |
+---------------------+
|   adk.Agent         |  <- Eino ADK ChatModelAgent
+---------------------+
```

### 14.3 与 einodoc/adk 的关系

```
+---------------------+
|  einodoc/adk/       |
|  - AgentFactory     |  <- 保留组合逻辑（SequentialAgent）
|  - Workflow         |
+---------------------+
          |
          | 调用 manager.GetAgent()
          ↓
+---------------------+
|  adkagents/         |
|  - Manager          |  <- 管理基础 Agent
|  - GetAgent()       |
+---------------------+
          |
          | 从 YAML 加载或缓存获取
          ↓
+---------------------+
|  agents/*.yaml      |  <- Agent 配置文件
+---------------------+
```

---

## 15. 约束条件

### 15.1 技术约束

- Agent 配置必须以 YAML 文件形式存放，不得硬编码
- Agent name 全局唯一，不可重复
- Agent 加载失败不应影响系统启动
- 必须适配 Eino ADK 的 `ChatModelAgentConfig` 结构
- 缓存的 ADK Agent 实例应在配置更新时失效

### 15.2 架构约束

- Manager 只负责基础 Agent 的装配，不负责组合逻辑
- 组合型 Agent（SequentialAgent 等）保持既有实现
- 不得修改 Eino ADK 的接口定义

### 15.3 性能约束

- 加载 20 个 Agent 配置应在 100ms 内完成
- `GetAgent()` 调用（命中缓存）应在 1ms 内完成
- 热加载检测间隔不超过 5 秒

---

## 16. 可修改 / 不可修改项

- ❌ 不可修改：
  - Eino ADK 的 `ChatModelAgentConfig` 结构
  - Agent name 唯一性约束
  - 基础 Agent 与组合 Agent 的职责划分

- ✅ 可调整：
  - 默认热加载间隔
  - 缓存策略（TTL、大小限制等）
  - 配置校验的严格程度
  - 错误处理方式

---

## 17. 风险与已知不确定点

| 风险点 | 说明 | 处理方式 |
|-------|------|---------|
| Agent 配置冲突 | 同名 Agent 多次定义 | 后加载的覆盖先加载的，记录警告 |
| Tool 不存在 | Tools 列表引用了不存在的工具 | 记录警告，跳过该工具 |
| 模型不存在 | 指定了不存在的模型 | 使用默认模型，记录警告 |
| 缓存失效 | 配置更新后缓存未失效 | 监听文件变更，自动清除缓存 |
| 热加载性能 | 频繁修改配置导致频繁重载 | 添加防抖机制，最小间隔 1 秒 |

---

## 18. 后续扩展方向（非本次需求）

- [ ] 支持 Agent 模板继承（基础模板 + 特定覆盖）
- [ ] 支持 Agent 参数化配置（如动态传入 basePath）
- [ ] 支持 Agent 版本管理
- [ ] 支持 Agent 组合逻辑的 YAML 配置化
- [ ] 支持 Agent 执行日志与监控
