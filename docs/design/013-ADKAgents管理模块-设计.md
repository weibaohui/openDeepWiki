# 013-ADKAgents管理模块-设计.md

## 1. 设计目标

实现基于 YAML 配置的 ADK Agent 管理模块，替代原有的硬编码 Agent 创建方式，支持热加载和动态扩展。

---

## 2. 模块结构

```
backend/internal/pkg/adkagents/
├── agent.go           # AgentDefinition 结构定义
├── manager.go         # Manager 实现（核心入口）
├── parser.go          # YAML 配置解析器
├── loader.go          # 配置加载器
├── registry.go        # Agent 定义注册表
├── watcher.go         # 文件变更监听（热加载）
├── errors.go          # 错误定义
├── types.go           # 公共类型定义
├── provider.go        # ModelProvider 和 ToolProvider 接口
└── README.md          # 使用文档
```

---

## 3. 核心组件设计

### 3.1 AgentDefinition（配置定义）

```go
type AgentDefinition struct {
    Name        string       `yaml:"name" json:"name"`
    Description string       `yaml:"description" json:"description"`
    Model       string       `yaml:"model" json:"model"`
    Instruction string       `yaml:"instruction" json:"instruction"`
    Tools       []string     `yaml:"tools" json:"tools"`
    MaxIterations int        `yaml:"maxIterations" json:"max_iterations"`
    Exit        ExitConfig   `yaml:"exit,omitempty" json:"exit,omitempty"`
    Path        string       `json:"path"`
    LoadedAt    time.Time    `json:"loaded_at"`
}
```

### 3.2 Manager（核心管理器）

```go
type Manager struct {
    config     *Config
    registry   *Registry
    cache      map[string]adk.Agent  // ADK Agent 实例缓存
    cacheMu    sync.RWMutex
    
    parser     *Parser
    loader     *Loader
    watcher    *FileWatcher
}

// 核心方法
func NewManager(config *Config) (*Manager, error)
func (m *Manager) GetAgent(name string) (adk.Agent, error)  // 核心接口
func (m *Manager) List() []*AgentDefinition
func (m *Manager) Reload(name string) error
func (m *Manager) Stop()
```

**缓存策略**：
- `GetAgent` 优先从缓存获取已创建的 ADK Agent 实例
- 配置变更时清除对应缓存，下次 `GetAgent` 重新创建
- 无缓存过期时间，依赖文件监听驱动更新

### 3.3 Provider 接口

```go
// ModelProvider 模型提供者
type ModelProvider interface {
    GetModel(name string) (model.ToolCallingChatModel, error)
    DefaultModel() model.ToolCallingChatModel
}

// ToolProvider 工具提供者  
type ToolProvider interface {
    GetTool(name string) (tool.BaseTool, error)
    ListTools() []string
}
```

**实现方案**：
- `ModelProvider` 由 `einodoc` 模块提供实现，复用现有的 LLM Client
- `ToolProvider` 由 `tools` 模块提供实现，从工具注册表获取

---

## 4. 关键流程

### 4.1 初始化流程

```
NewManager(config)
    ↓
解析配置目录（环境变量 > 配置 > 默认）
    ↓
创建 Registry、Parser、Loader
    ↓
LoadFromDir(dir) 加载所有 YAML 配置
    ↓
如果 AutoReload，启动 FileWatcher
    ↓
返回 Manager 实例
```

### 4.2 GetAgent 流程

```
GetAgent(name)
    ↓
从 Registry 获取 AgentDefinition
    ↓
检查缓存（cache[name]）
    ├── 存在 → 返回缓存的 adk.Agent
    └── 不存在 → 继续创建
        ↓
通过 ModelProvider 获取 ChatModel
    ↓
通过 ToolProvider 获取 Tools
    ↓
构造 adk.ChatModelAgentConfig
    ↓
调用 adk.NewChatModelAgent() 创建
    ↓
存入缓存
    ↓
返回 adk.Agent
```

### 4.3 热加载流程

```
FileWatcher 检测到文件变更
    ↓
根据事件类型处理：
    ├── Create → Loader.LoadFromPath(path)
    ├── Modify → Loader.Reload(name)
    └── Delete → Loader.Unload(name)
        ↓
更新 Registry
    ↓
清除对应缓存（如果存在）
    ↓
记录日志
```

---

## 5. 错误处理

```go
var (
    ErrAgentNotFound    = errors.New("agent not found")
    ErrInvalidConfig    = errors.New("invalid agent config")
    ErrInvalidName      = errors.New("invalid agent name")
    ErrToolNotFound     = errors.New("tool not found")
    ErrModelNotFound    = errors.New("model not found")
    ErrAgentDirNotFound = errors.New("agents directory not found")
)
```

**处理原则**：
- 单个 Agent 加载失败不影响其他 Agent
- 工具不存在记录警告，跳过该工具，不阻断 Agent 加载
- 模型不存在时使用默认模型

---

## 6. 与原系统的集成

### 6.1 ModelProvider 实现

在 `backend/internal/service/einodoc/adk/` 中实现：

```go
// modelProvider 实现 adkagents.ModelProvider
type modelProvider struct {
    chatModel model.ToolCallingChatModel
}

func (p *modelProvider) GetModel(name string) (model.ToolCallingChatModel, error) {
    if name == "" {
        return p.chatModel, nil
    }
    // TODO: 支持多模型时扩展
    return p.chatModel, nil
}

func (p *modelProvider) DefaultModel() model.ToolCallingChatModel {
    return p.chatModel
}
```

### 6.2 ToolProvider 实现

复用现有的工具注册机制：

```go
// toolProvider 实现 adkagents.ToolProvider
type toolProvider struct {
    basePath string
}

func (p *toolProvider) GetTool(name string) (tool.BaseTool, error) {
    switch name {
    case "list_dir":
        return tools.NewListDirTool(p.basePath), nil
    case "read_file":
        return tools.NewReadFileTool(p.basePath), nil
    case "search_files":
        return tools.NewSearchFilesTool(p.basePath), nil
    default:
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
}
```

### 6.3 AgentFactory 改造

```go
type AgentFactory struct {
    manager  *adkagents.Manager
    basePath string
}

func NewAgentFactory(chatModel model.ToolCallingChatModel, basePath string) (*AgentFactory, error) {
    // 创建 providers
    modelProvider := &modelProvider{chatModel: chatModel}
    toolProvider := &toolProvider{basePath: basePath}
    
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
        return nil, err
    }
    
    return &AgentFactory{
        manager:  manager,
        basePath: basePath,
    }, nil
}

// CreateSequentialAgent 保持既有逻辑
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
    initializer, _ := f.manager.GetAgent(AgentRepoInitializer)
    architect, _ := f.manager.GetAgent(AgentArchitect)
    // ... 其他 Agent
    
    return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
        SubAgents: []adk.Agent{initializer, architect, ...},
    })
}
```

---

## 7. YAML 配置规范

### 7.1 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Agent 唯一标识，符合 [a-z0-9-]+ |
| description | string | 是 | Agent 描述 |
| model | string | 否 | 模型名称，空则使用默认 |
| instruction | string | 是 | System Prompt |
| tools | []string | 否 | 工具名称列表 |
| maxIterations | int | 是 | 最大迭代次数 |
| exit | object | 否 | 退出条件配置 |

### 7.2 完整示例

```yaml
name: Explorer
description: 代码探索者 - 负责深度分析代码结构和依赖关系

instruction: |
  你的任务是深入探索代码库：
  1. 读取 README 和关键配置文件
  2. 搜索核心代码文件
  3. 分析项目的主要模块和组件
  
  每一轮探索完成后，请明确说明探索进度。

tools:
  - search_files
  - list_dir
  - read_file

maxIterations: 15
```

---

## 8. 测试策略

### 8.1 单元测试

- `parser_test.go`: 解析和校验逻辑
- `registry_test.go`: 注册表 CRUD 操作
- `loader_test.go`: 加载器功能
- `manager_test.go`: Manager 核心功能

### 8.2 集成测试

- 与 ModelProvider、ToolProvider 集成
- 热加载功能测试
- 缓存失效测试

---

## 9. 实施计划

1. **创建 adkagents 模块基础结构**（agent.go, errors.go, types.go）
2. **实现配置解析器**（parser.go）
3. **实现注册表**（registry.go）
4. **实现加载器**（loader.go）
5. **实现 Manager**（manager.go）
6. **实现文件监听器**（watcher.go）
7. **调整 YAML 配置文件**（agents/*.yaml）
8. **改造 AgentFactory**（agents.go）
9. **编写测试和文档**
