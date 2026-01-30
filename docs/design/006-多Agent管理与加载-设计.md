# 006-多Agent管理与加载-设计.md

## 1. 架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LLM 会话执行流程                                 │
│                                                                              │
│   用户请求 ──► Agent Router ──► Agent Manager ──► Agent                      │
│                                    │              │                          │
│                                    │              ▼                          │
│                                    │         ┌─────────┐                     │
│                                    │         │  Agent  │                     │
│                                    │         │ - name  │                     │
│                                    │         │ - systemPrompt              │
│                                    │         │ - mcpPolicy                 │
│                                    │         │ - skillPolicy               │
│                                    │         │ - runtimePolicy             │
│                                    │         └────┬────┘                     │
│                                    │              │                          │
│                                    ▼              ▼                          │
│                            ┌─────────────────────────────┐                  │
│                            │      构造 LLM 上下文         │                  │
│                            │  - system prompt            │                  │
│                            │  - available tools          │                  │
│                            │  - mcp context              │                  │
│                            └─────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           Agent 管理组件架构                                  │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Agent Router                                │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐ │   │
│  │  │  Explicit Name  │  │  EntryPoint     │  │  Default Agent      │ │   │
│  │  │  (显式指定)      │  │  (入口路由)      │  │  (默认兜底)          │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Agent Manager                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │   Parser    │  │   Loader    │  │  Registry   │  │  Watcher   │ │   │
│  │  │ (配置解析)   │  │ (加载器)     │  │ (注册中心)   │  │ (热加载)    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                         │
│                                    ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Agent Registry                               │   │
│  │                    (name -> *Agent 映射)                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                         │
│          ┌─────────────────────────┼─────────────────────────┐              │
│          │                         │                         │              │
│          ▼                         ▼                         ▼              │
│  ┌───────────────┐        ┌───────────────┐        ┌───────────────┐       │
│  │   agents/     │        │   agents/     │        │   agents/     │       │
│  │  diagnose-    │        │    ops-       │        │   default-    │       │
│  │   agent.yaml  │        │   agent.yaml  │        │   agent.yaml  │       │
│  └───────────────┘        └───────────────┘        └───────────────┘       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 数据流

```
1. 系统启动
   │
   ├── 解析 Agents 目录
   │
   ├── 遍历每个 .yaml/.json 文件
   │   ├── 读取配置文件
   │   ├── 解析 YAML/JSON
   │   ├── 校验 Agent 定义
   │   ├── 校验 MCP Policy 引用
   │   ├── 校验 Skill Policy 引用
   │   └── 注册到 Registry
   │
   └── 启动 File Watcher（热加载）

2. 路由选择
   │
   ├── 接收请求（带上下文）
   │
   ├── Router 判断选择策略（优先级）
   │   ├── 1. 显式指定 Agent name
   │   ├── 2. EntryPoint 匹配
   │   └── 3. 默认 Agent 兜底
   │
   ├── 从 Registry 获取 Agent
   │
   └── 返回选中的 Agent

3. 会话执行
   │
   ├── 获取 Agent 定义
   │
   ├── 根据 MCP Policy 获取上下文
   │   └── 调用允许的 MCP 获取数据
   │
   ├── 根据 Skill Policy 构造可用 tools
   │   └── 过滤 allow/deny 列表
   │
   ├── 构造 system prompt
   │
   ├── 应用 Runtime Policy 约束
   │   └── maxSteps, riskLevel, requireConfirmation
   │
   └── 调用 LLM 执行会话
```

---

## 2. 核心组件设计

### 2.1 Agent 结构

```go
package agents

import (
    "time"
)

// Agent Agent定义
type Agent struct {
    // 元数据
    Name        string `yaml:"name" json:"name"`
    Version     string `yaml:"version" json:"version"`
    Description string `yaml:"description" json:"description"`

    // System Prompt
    SystemPrompt string `yaml:"systemPrompt" json:"system_prompt"`

    // MCP Policy
    McpPolicy McpPolicy `yaml:"mcp" json:"mcp_policy"`

    // Skill Policy
    SkillPolicy SkillPolicy `yaml:"skills" json:"skill_policy"`

    // Runtime Policy
    RuntimePolicy RuntimePolicy `yaml:"policies" json:"runtime_policy"`

    // 路径信息
    Path     string    `json:"path"` // 配置文件路径
    LoadedAt time.Time `json:"loaded_at"`
}

// McpPolicy MCP策略
type McpPolicy struct {
    Allowed  []string `yaml:"allowed" json:"allowed"`    // 允许的 MCP 列表
    MaxCalls int      `yaml:"maxCalls" json:"max_calls"` // 最大调用次数
}

// SkillPolicy Skill策略
type SkillPolicy struct {
    Allow []string `yaml:"allow" json:"allow"` // 显式允许的 Skills
    Deny  []string `yaml:"deny" json:"deny"`   // 显式禁止的 Skills
}

// RuntimePolicy 运行时策略
type RuntimePolicy struct {
    RiskLevel           string `yaml:"riskLevel" json:"risk_level"`                     // 风险等级：read / write / admin
    MaxSteps            int    `yaml:"maxSteps" json:"max_steps"`                       // 最大执行步骤数
    RequireConfirmation bool   `yaml:"requireConfirmation" json:"require_confirmation"` // 是否需要确认
}
```

### 2.2 Router Context 结构

```go
package agents

// RouterContext 路由上下文
type RouterContext struct {
    AgentName  string            // 显式指定的 Agent name
    EntryPoint string            // 用户入口（如 "diagnose", "ops"）
    TaskType   string            // 任务类型
    Metadata   map[string]string // 附加元数据
}
```

### 2.3 Agent Parser

```go
package agents

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "gopkg.in/yaml.v3"
)

// Parser Agent 配置解析器
type Parser struct {
    maxDescriptionLen int
    maxNameLen        int
}

// NewParser 创建解析器
func NewParser() *Parser {
    return &Parser{
        maxDescriptionLen: 1024,
        maxNameLen:        64,
    }
}

// Parse 解析 Agent 配置文件
func (p *Parser) Parse(configPath string) (*Agent, error) {
    configPath = filepath.Clean(configPath)

    // 读取文件内容
    content, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read agent config: %w", err)
    }

    // 解析 YAML
    agent := &Agent{}
    if err := yaml.Unmarshal(content, agent); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
    }

    // 设置路径信息
    agent.Path = configPath
    agent.LoadedAt = time.Now()

    // 校验
    if err := p.Validate(agent); err != nil {
        return nil, err
    }

    return agent, nil
}

// Validate 校验 Agent 配置
func (p *Parser) Validate(agent *Agent) error {
    // 校验 name
    if agent.Name == "" {
        return fmt.Errorf("%w: name is required", ErrInvalidName)
    }
    if len(agent.Name) > p.maxNameLen {
        return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidName, p.maxNameLen)
    }
    if !isValidAgentName(agent.Name) {
        return fmt.Errorf("%w: name must contain only lowercase letters, numbers, and hyphens", ErrInvalidName)
    }

    // 校验 version
    if agent.Version == "" {
        return fmt.Errorf("%w: version is required", ErrInvalidConfig)
    }
    if !isValidVersion(agent.Version) {
        return fmt.Errorf("%w: version must be valid semantic version", ErrInvalidConfig)
    }

    // 校验 description
    if agent.Description == "" {
        return fmt.Errorf("%w: description is required", ErrInvalidConfig)
    }
    if len(agent.Description) > p.maxDescriptionLen {
        return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidConfig, p.maxDescriptionLen)
    }

    // 校验 systemPrompt
    if agent.SystemPrompt == "" {
        return fmt.Errorf("%w: systemPrompt is required", ErrInvalidConfig)
    }

    // 校验 riskLevel
    if agent.RuntimePolicy.RiskLevel != "" {
        validRiskLevels := map[string]bool{"read": true, "write": true, "admin": true}
        if !validRiskLevels[agent.RuntimePolicy.RiskLevel] {
            return fmt.Errorf("%w: riskLevel must be one of: read, write, admin", ErrInvalidConfig)
        }
    }

    return nil
}

// isValidAgentName 校验 name 格式
func isValidAgentName(name string) bool {
    if name == "" {
        return false
    }
    // 不能以连字符开头或结尾
    if name[0] == '-' || name[len(name)-1] == '-' {
        return false
    }
    // 不能包含连续连字符
    if strings.Contains(name, "--") {
        return false
    }
    // 只能包含小写字母、数字、连字符
    validPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
    return validPattern.MatchString(name)
}

// isValidVersion 校验 version 格式（简单语义化版本）
func isValidVersion(version string) bool {
    // 支持 v1, v1.0, v1.0.0 格式
    pattern := regexp.MustCompile(`^v\d+(\.\d+)?(\.\d+)?$`)
    return pattern.MatchString(version)
}
```

### 2.4 Agent Registry

```go
package agents

import (
    "fmt"
    "sync"
    "time"
)

// Registry Agent 注册中心接口
type Registry interface {
    // Register 注册 Agent
    Register(agent *Agent) error

    // Unregister 注销 Agent
    Unregister(name string) error

    // Get 获取指定名称的 Agent
    Get(name string) (*Agent, error)

    // List 列出所有已注册的 Agents
    List() []*Agent

    // Exists 检查 Agent 是否存在
    Exists(name string) bool
}

// registry Registry 的实现
type registry struct {
    mu     sync.RWMutex
    agents map[string]*Agent // name -> Agent
}

// NewRegistry 创建新的 Registry 实例
func NewRegistry() Registry {
    return &registry{
        agents: make(map[string]*Agent),
    }
}

// Register 注册 Agent
func (r *registry) Register(agent *Agent) error {
    if agent == nil {
        return fmt.Errorf("agent cannot be nil")
    }

    name := agent.Name
    if name == "" {
        return fmt.Errorf("agent name cannot be empty")
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    r.agents[name] = agent
    return nil
}

// Unregister 注销 Agent
func (r *registry) Unregister(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.agents[name]; !exists {
        return fmt.Errorf("%w: %s", ErrAgentNotFound, name)
    }

    delete(r.agents, name)
    return nil
}

// Get 获取指定名称的 Agent
func (r *registry) Get(name string) (*Agent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    agent, exists := r.agents[name]
    if !exists {
        return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, name)
    }

    return agent, nil
}

// List 列出所有已注册的 Agents
func (r *registry) List() []*Agent {
    r.mu.RLock()
    defer r.mu.RUnlock()

    result := make([]*Agent, 0, len(r.agents))
    for _, agent := range r.agents {
        result = append(result, agent)
    }

    return result
}

// Exists 检查 Agent 是否存在
func (r *registry) Exists(name string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    _, exists := r.agents[name]
    return exists
}
```

### 2.5 Agent Loader

```go
package agents

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

// Loader Agent 加载器
type Loader struct {
    parser   *Parser
    registry Registry
    mu       sync.RWMutex
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry Registry) *Loader {
    return &Loader{
        parser:   parser,
        registry: registry,
    }
}

// LoadFromDir 从目录加载所有 Agents
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error) {
    dir = filepath.Clean(dir)

    // 检查目录是否存在
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        log.Printf("Agents directory does not exist: %s", dir)
        return nil, nil
    }

    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, fmt.Errorf("failed to read agents directory: %w", err)
    }

    results := make([]*LoadResult, 0)

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        // 只处理 .yaml 和 .json 文件
        ext := strings.ToLower(filepath.Ext(entry.Name()))
        if ext != ".yaml" && ext != ".yml" && ext != ".json" {
            continue
        }

        configPath := filepath.Join(dir, entry.Name())
        result := l.loadAgent(configPath)
        results = append(results, result)
    }

    return results, nil
}

// LoadFromPath 加载单个 Agent
func (l *Loader) LoadFromPath(path string) (*Agent, error) {
    result := l.loadAgent(path)
    if result.Error != nil {
        return nil, result.Error
    }
    return result.Agent, nil
}

// loadAgent 加载 Agent（内部）
func (l *Loader) loadAgent(path string) *LoadResult {
    agent, err := l.parser.Parse(path)
    if err != nil {
        return &LoadResult{
            Error:  err,
            Action: "failed",
        }
    }

    // 检查是否已存在
    existing, _ := l.registry.Get(agent.Name)
    action := "created"
    if existing != nil {
        action = "updated"
    }

    // 注册到 Registry
    if err := l.registry.Register(agent); err != nil {
        return &LoadResult{
            Agent:  agent,
            Error:  err,
            Action: "failed",
        }
    }

    return &LoadResult{
        Agent:  agent,
        Action: action,
    }
}

// Unload 卸载 Agent
func (l *Loader) Unload(name string) error {
    return l.registry.Unregister(name)
}

// Reload 重新加载 Agent
func (l *Loader) Reload(name string) (*Agent, error) {
    agent, err := l.registry.Get(name)
    if err != nil {
        return nil, err
    }

    l.Unload(name)
    return l.LoadFromPath(agent.Path)
}

// LoadResult 加载结果
type LoadResult struct {
    Agent  *Agent
    Error  error
    Action string // "created", "updated", "failed"
}
```

### 2.6 Agent Router

```go
package agents

import (
    "fmt"
)

// Router Agent 路由器接口
type Router interface {
    // Route 根据上下文选择 Agent
    Route(ctx RouterContext) (*Agent, error)

    // SetDefault 设置默认 Agent
    SetDefault(agentName string) error

    // RegisterRoute 注册路由规则
    RegisterRoute(entryPoint string, agentName string)
}

// router Router 的实现
type router struct {
    registry     Registry
    defaultAgent string
    routes       map[string]string // entryPoint -> agentName
}

// NewRouter 创建新的 Router 实例
func NewRouter(registry Registry) Router {
    return &router{
        registry: registry,
        routes:   make(map[string]string),
    }
}

// Route 根据上下文选择 Agent
func (r *router) Route(ctx RouterContext) (*Agent, error) {
    // 1. 优先级最高：显式指定 Agent name
    if ctx.AgentName != "" {
        agent, err := r.registry.Get(ctx.AgentName)
        if err != nil {
            return nil, fmt.Errorf("explicitly specified agent not found: %w", err)
        }
        return agent, nil
    }

    // 2. 根据 EntryPoint 路由
    if ctx.EntryPoint != "" {
        if agentName, exists := r.routes[ctx.EntryPoint]; exists {
            agent, err := r.registry.Get(agentName)
            if err != nil {
                return nil, fmt.Errorf("route found but agent not found: %w", err)
            }
            return agent, nil
        }
    }

    // 3. 使用默认 Agent
    if r.defaultAgent != "" {
        agent, err := r.registry.Get(r.defaultAgent)
        if err != nil {
            return nil, fmt.Errorf("default agent not found: %w", err)
        }
        return agent, nil
    }

    // 4. 没有任何可用 Agent
    return nil, fmt.Errorf("%w: no matching agent found", ErrAgentNotFound)
}

// SetDefault 设置默认 Agent
func (r *router) SetDefault(agentName string) error {
    if !r.registry.Exists(agentName) {
        return fmt.Errorf("%w: %s", ErrAgentNotFound, agentName)
    }
    r.defaultAgent = agentName
    return nil
}

// RegisterRoute 注册路由规则
func (r *router) RegisterRoute(entryPoint string, agentName string) {
    r.routes[entryPoint] = agentName
}
```

### 2.7 Agent Manager（整合）

```go
package agents

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"
)

// Config Manager 配置
type Config struct {
    Dir            string
    AutoReload     bool
    ReloadInterval time.Duration
    DefaultAgent   string
    Routes         map[string]string // entryPoint -> agentName
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
    return &Config{
        Dir:            "./agents",
        AutoReload:     true,
        ReloadInterval: 5 * time.Second,
        Routes:         make(map[string]string),
    }
}

// Manager Agent 管理器
type Manager struct {
    Config   *Config
    Registry Registry
    Parser   *Parser
    Loader   *Loader
    Router   Router
    watcher  *FileWatcher
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error) {
    if config == nil {
        config = DefaultConfig()
    }

    // 解析目录
    dir, err := resolveAgentsDir(config.Dir)
    if err != nil {
        return nil, err
    }
    config.Dir = dir

    log.Printf("Agents directory: %s", dir)

    // 确保目录存在
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create agents directory: %w", err)
    }

    // 创建组件
    registry := NewRegistry()
    parser := NewParser()
    loader := NewLoader(parser, registry)
    router := NewRouter(registry)

    // 注册路由规则
    for entryPoint, agentName := range config.Routes {
        router.RegisterRoute(entryPoint, agentName)
    }

    m := &Manager{
        Config:   config,
        Registry: registry,
        Parser:   parser,
        Loader:   loader,
        Router:   router,
    }

    // 初始加载
    results, err := loader.LoadFromDir(dir)
    if err != nil {
        log.Printf("Warning: failed to load agents: %v", err)
    } else {
        loaded := 0
        updated := 0
        failed := 0
        for _, r := range results {
            switch r.Action {
            case "created":
                loaded++
            case "updated":
                updated++
            case "failed":
                failed++
                log.Printf("Failed to load agent from %s: %v", r.Agent.Path, r.Error)
            }
        }
        if loaded > 0 || updated > 0 {
            log.Printf("Loaded %d agents, updated %d agents", loaded, updated)
        }
        if failed > 0 {
            log.Printf("Failed to load %d agents", failed)
        }
    }

    // 设置默认 Agent
    if config.DefaultAgent != "" {
        if err := router.SetDefault(config.DefaultAgent); err != nil {
            log.Printf("Warning: failed to set default agent: %v", err)
        }
    }

    // 启动热加载
    if config.AutoReload {
        m.startWatcher()
    }

    return m, nil
}

// startWatcher 启动文件监听
func (m *Manager) startWatcher() {
    m.watcher = NewFileWatcher(m.Config.Dir, m.Config.ReloadInterval, func(event FileEvent) {
        switch event.Type {
        case "create":
            log.Printf("Loading new agent from %s", event.Path)
            if _, err := m.Loader.LoadFromPath(event.Path); err != nil {
                log.Printf("Failed to load agent: %v", err)
            } else {
                log.Printf("Successfully loaded agent from %s", event.Path)
            }

        case "modify":
            agentName := guessAgentNameFromPath(event.Path)
            log.Printf("Reloading agent: %s", agentName)
            if _, err := m.Loader.Reload(agentName); err != nil {
                log.Printf("Failed to reload agent: %v", err)
            } else {
                log.Printf("Successfully reloaded agent: %s", agentName)
            }

        case "delete":
            agentName := guessAgentNameFromPath(event.Path)
            log.Printf("Unloading agent: %s", agentName)
            if err := m.Loader.Unload(agentName); err != nil {
                log.Printf("Failed to unload agent: %v", err)
            } else {
                log.Printf("Successfully unloaded agent: %s", agentName)
            }
        }
    })

    if err := m.watcher.Start(); err != nil {
        log.Printf("Warning: failed to start file watcher: %v", err)
    }
}

// Stop 停止 Manager
func (m *Manager) Stop() {
    if m.watcher != nil {
        m.watcher.Stop()
    }
}

// SelectAgent 根据上下文选择 Agent
func (m *Manager) SelectAgent(ctx RouterContext) (*Agent, error) {
    return m.Router.Route(ctx)
}

// ReloadAll 重新加载所有 Agents
func (m *Manager) ReloadAll() error {
    // 获取当前所有 agents
    agents := m.Registry.List()

    // 卸载所有
    for _, agent := range agents {
        if err := m.Loader.Unload(agent.Name); err != nil {
            log.Printf("Failed to unload agent %s: %v", agent.Name, err)
        }
    }

    // 重新加载
    _, err := m.Loader.LoadFromDir(m.Config.Dir)
    return err
}

// resolveAgentsDir 解析 Agents 目录
func resolveAgentsDir(configDir string) (string, error) {
    // 1. 环境变量
    if dir := os.Getenv("AGENTS_DIR"); dir != "" {
        return filepath.Abs(dir)
    }

    // 2. 配置
    if configDir != "" {
        return filepath.Abs(configDir)
    }

    // 3. 默认
    exePath, err := os.Executable()
    if err != nil {
        cwd, _ := os.Getwd()
        return filepath.Join(cwd, "agents"), nil
    }
    return filepath.Join(filepath.Dir(exePath), "agents"), nil
}

// guessAgentNameFromPath 从路径猜测 Agent name
func guessAgentNameFromPath(path string) string {
    // 从文件名提取（去掉扩展名）
    base := filepath.Base(path)
    ext := filepath.Ext(base)
    return strings.TrimSuffix(base, ext)
}
```

---

## 3. 示例 Agent 配置

### 3.1 诊断 Agent（diagnose-agent.yaml）

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

### 3.2 运维 Agent（ops-agent.yaml）

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

## 4. 错误定义

```go
package agents

import "errors"

// 预定义错误
var (
    // ErrAgentNotFound Agent 不存在
    ErrAgentNotFound = errors.New("agent not found")

    // ErrInvalidConfig 配置无效
    ErrInvalidConfig = errors.New("invalid agent config")

    // ErrInvalidName name 格式错误
    ErrInvalidName = errors.New("invalid agent name")

    // ErrAgentLoadFailed 加载失败
    ErrAgentLoadFailed = errors.New("failed to load agent")

    // ErrAgentDirNotFound Agents 目录不存在
    ErrAgentDirNotFound = errors.New("agents directory not found")

    // ErrConfigNotFound 配置文件不存在
    ErrConfigNotFound = errors.New("agent config file not found")
)
```

---

## 5. 文件监听（Watcher）

复用 skills 包的 FileWatcher，或创建类似的实现：

```go
package agents

import (
    "os"
    "path/filepath"
    "strings"
    "time"
)

// FileEvent 文件事件
type FileEvent struct {
    Type string // "create", "modify", "delete"
    Path string
}

// FileWatcher 文件监听器
type FileWatcher struct {
    dir      string
    interval time.Duration
    callback func(FileEvent)
    stop     chan bool
    states   map[string]os.FileInfo
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher(dir string, interval time.Duration, callback func(FileEvent)) *FileWatcher {
    return &FileWatcher{
        dir:      dir,
        interval: interval,
        callback: callback,
        stop:     make(chan bool),
        states:   make(map[string]os.FileInfo),
    }
}

// Start 启动监听
func (w *FileWatcher) Start() error {
    go w.watch()
    return nil
}

// Stop 停止监听
func (w *FileWatcher) Stop() {
    close(w.stop)
}

// watch 监听循环
func (w *FileWatcher) watch() {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    // 初始状态
    w.scan()

    for {
        select {
        case <-ticker.C:
            w.scan()
        case <-w.stop:
            return
        }
    }
}

// scan 扫描目录变化
func (w *FileWatcher) scan() {
    entries, err := os.ReadDir(w.dir)
    if err != nil {
        return
    }

    current := make(map[string]os.FileInfo)

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        // 只监控 .yaml, .yml, .json 文件
        ext := strings.ToLower(filepath.Ext(entry.Name()))
        if ext != ".yaml" && ext != ".yml" && ext != ".json" {
            continue
        }

        path := filepath.Join(w.dir, entry.Name())
        info, err := entry.Info()
        if err != nil {
            continue
        }

        current[path] = info

        // 检查变化
        if old, exists := w.states[path]; !exists {
            // 新建
            w.callback(FileEvent{Type: "create", Path: path})
        } else if info.ModTime() != old.ModTime() || info.Size() != old.Size() {
            // 修改
            w.callback(FileEvent{Type: "modify", Path: path})
        }
    }

    // 检查删除
    for path := range w.states {
        if _, exists := current[path]; !exists {
            w.callback(FileEvent{Type: "delete", Path: path})
        }
    }

    w.states = current
}
```

---

## 6. 代码目录结构

```
backend/internal/pkg/agents/
├── agent.go              # Agent 结构定义
├── registry.go           # Registry 接口与实现
├── parser.go             # 配置解析器
├── loader.go             # Agent 加载器
├── router.go             # Agent 路由器
├── manager.go            # 管理器（整合）
├── watcher.go            # 文件监听（热加载）
├── errors.go             # 错误定义
├── types.go              # 公共类型（RouterContext, LoadResult 等）
├── parser_test.go        # 解析器测试
├── loader_test.go        # 加载器测试
├── router_test.go        # 路由器测试
└── README.md             # 使用文档

agents/                   # Agent 配置目录
├── diagnose-agent.yaml   # 诊断 Agent
├── ops-agent.yaml        # 运维 Agent
└── default-agent.yaml    # 默认 Agent
```

---

## 7. 使用示例

```go
package main

import (
    "log"
    
    "github.com/opendeepwiki/backend/internal/pkg/agents"
)

func main() {
    // 创建 Manager
    config := &agents.Config{
        Dir:          "./agents",
        AutoReload:   true,
        DefaultAgent: "default-agent",
        Routes: map[string]string{
            "diagnose": "diagnose-agent",
            "ops":      "ops-agent",
        },
    }

    manager, err := agents.NewManager(config)
    if err != nil {
        log.Fatalf("Failed to create agent manager: %v", err)
    }
    defer manager.Stop()

    // 路由选择 Agent
    ctx := agents.RouterContext{
        EntryPoint: "diagnose", // 用户访问诊断页面
    }

    agent, err := manager.SelectAgent(ctx)
    if err != nil {
        log.Fatalf("Failed to select agent: %v", err)
    }

    log.Printf("Selected agent: %s", agent.Name)
    log.Printf("System Prompt: %s", agent.SystemPrompt)
    log.Printf("Allowed Skills: %v", agent.SkillPolicy.Allow)
    log.Printf("Risk Level: %s", agent.RuntimePolicy.RiskLevel)
}
```

---

## 8. 测试策略

### 8.1 Parser 测试

- [ ] 正常解析 YAML 配置
- [ ] 缺少必需字段（name, version, description, systemPrompt）
- [ ] name 格式错误
- [ ] version 格式错误
- [ ] riskLevel 无效值
- [ ] YAML 语法错误

### 8.2 Registry 测试

- [ ] 注册 Agent
- [ ] 注销 Agent
- [ ] 获取存在的 Agent
- [ ] 获取不存在的 Agent
- [ ] 列出所有 Agents
- [ ] 并发安全测试

### 8.3 Loader 测试

- [ ] 加载目录下所有 Agents
- [ ] 加载单个 Agent
- [ ] 重新加载更新
- [ ] 卸载 Agent
- [ ] 无效配置文件处理

### 8.4 Router 测试

- [ ] 显式指定 Agent name
- [ ] EntryPoint 路由匹配
- [ ] 默认 Agent 兜底
- [ ] 未找到 Agent 错误
- [ ] 设置默认 Agent

### 8.5 Manager 集成测试

- [ ] 完整流程：加载 -> 路由 -> 获取 Agent
- [ ] 热加载测试
- [ ] 并发安全测试

---

## 9. 与 Skills 的关系

Agent 包与 Skills 包的关系：

```
┌─────────────────────────────────────────────────────────────┐
│                         Agent                               │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  SkillPolicy                                          │ │
│  │  - Allow: ["search_logs", "analyze_logs"]             │ │
│  │  - Deny:  ["restart_pod"]                             │ │
│  └───────────────────────────────────────────────────────┘ │
│                            │                                │
│                            ▼                                │
│  ┌───────────────────────────────────────────────────────┐ │
│  │              Skill Registry (skills pkg)              │ │
│  │  - search_logs:  ✓ 可用                               │ │
│  │  - analyze_logs: ✓ 可用                               │ │
│  │  - restart_pod:  ✗ 被 Agent 禁止                      │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

Agent 通过 SkillPolicy 从全局 Skill Registry 中筛选可用 Skills，构造 LLM tools 列表。
