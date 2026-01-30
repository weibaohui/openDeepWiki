# 005-Skills动态加载-设计.md

## 1. 设计概述

本文档详细描述 Skills 动态加载功能的实现方案。

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        LLM Client                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              ChatWithToolExecution                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 Registry.ToTools()                  │   │
│  │         (将 enabled Skills 转为 LLM Tools)          │   │
│  └─────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                   Skill Registry (单例)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   skills    │  │   status    │  │      mutex          │ │
│  │  (map[name] │  │  (map[name] │  │   (RWMutex)         │ │
│  │   Skill)    │  │   bool)     │  │                     │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│  - Register()    - Enable()                                │
│  - Unregister()  - Disable()                               │
│  - Get()         - List()                                  │
│  - ListEnabled() - ToTools()                               │
└───────────────────────────┬─────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│   Builtin     │   │     HTTP      │   │   (Future)    │
│   Provider    │   │   Provider    │   │ WASM Provider │
├───────────────┤   ├───────────────┤   ├───────────────┤
│ Go 代码实现   │   │ HTTP 调用     │   │ WASM 运行时   │
│ 直接注册      │   │ 远程服务      │   │ 沙箱执行      │
└───────────────┘   └───────────────┘   └───────────────┘
        ▲
        │
┌───────┴─────────────────────────────────────────────────┐
│                    Config Loader                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   Load      │  │   Watch     │  │    Validate     │  │
│  │  初始加载    │  │  文件监听    │  │    配置校验      │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
│                                                         │
│  配置来源（优先级从高到低）：                              │
│  1. SKILLS_DIR 环境变量                                  │
│  2. config.yaml 中 skills.dir                            │
│  3. 默认 ./skills 目录                                   │
└─────────────────────────────────────────────────────────┘
```

### 2.2 数据流

```
1. 系统启动
   │
   ├── 加载配置目录（./skills 或环境变量指定）
   │
   ├── 解析所有 YAML 配置文件
   │
   ├── 根据 provider 类型创建对应 Provider
   │
   ├── Provider 创建 Skill 实例
   │
   └── Registry.Register(skill)

2. 运行时变更
   │
   ├── 文件监听器检测到变更
   │
   ├── 重新加载配置文件
   │
   ├── 对比新旧配置
   │
   └── 执行 Register/Unregister/Update

3. LLM 调用流程
   │
   ├── LLM Client 调用 Registry.ToTools()
   │
   ├── 获取所有 enabled Skills 的 Tool 定义
   │
   ├── 发送给 LLM
   │
   ├── LLM 返回 tool_calls
   │
   ├── 根据 name 从 Registry 获取 Skill
   │
   ├── 调用 Skill.Execute(ctx, args)
   │
   └── 返回结果给 LLM
```

---

## 3. 核心接口设计

### 3.1 Skill 接口

```go
package skills

import (
    "context"
    "encoding/json"
)

// Skill 技能接口，所有技能必须实现
type Skill interface {
    // Name 返回技能唯一名称
    // 约束：全局唯一，符合 [a-zA-Z0-9_-]+ 格式
    Name() string
    
    // Description 返回技能描述
    // 供 LLM 理解该技能的用途
    Description() string
    
    // Parameters 返回参数 JSON Schema
    // 符合 JSON Schema Draft 7 规范
    Parameters() ParameterSchema
    
    // Execute 执行技能
    // ctx: 上下文，包含超时控制
    // args: JSON 格式的参数，需根据 Parameters 解析
    // 返回: 执行结果（必须可 JSON 序列化）和错误
    Execute(ctx context.Context, args json.RawMessage) (interface{}, error)
    
    // ProviderType 返回提供者类型
    // 用于调试和监控
    ProviderType() string
}

// ParameterSchema 参数 JSON Schema 定义
type ParameterSchema struct {
    Type       string              `json:"type"`                 // 固定为 "object"
    Properties map[string]Property `json:"properties"`           // 参数属性
    Required   []string            `json:"required,omitempty"`   // 必需参数列表
}

// Property 单个参数属性
type Property struct {
    Type        string      `json:"type"`                   // string, integer, boolean, array, object
    Description string      `json:"description"`            // 参数描述
    Enum        []string    `json:"enum,omitempty"`         // 可选的枚举值
    Items       *Property   `json:"items,omitempty"`        // 数组元素类型（当 type 为 array）
}
```

### 3.2 Registry 接口与实现

```go
package skills

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/weibh/openDeepWiki/backend/internal/pkg/llm"
)

// Registry Skill 注册中心接口
type Registry interface {
    // Register 注册 Skill
    // 如果同名 Skill 已存在，返回错误
    Register(skill Skill) error
    
    // Unregister 注销 Skill
    // 如果不存在，返回 ErrSkillNotFound
    Unregister(name string) error
    
    // Enable 启用 Skill
    // 只有 enabled 的 Skill 才会被暴露给 LLM
    Enable(name string) error
    
    // Disable 禁用 Skill
    // 禁用后 Skill 保留在 Registry 中，但不可被 LLM 调用
    Disable(name string) error
    
    // Get 获取指定名称的 Skill
    Get(name string) (Skill, error)
    
    // List 列出所有已注册的 Skills
    List() []Skill
    
    // ListEnabled 列出所有已启用的 Skills
    ListEnabled() []Skill
    
    // ToTools 将所有 enabled Skills 转换为 LLM Tools
    ToTools() []llm.Tool
}

// registry Registry 的实现
type registry struct {
    mu       sync.RWMutex
    skills   map[string]Skill    // name -> Skill
    enabled  map[string]bool     // name -> enabled
}

// NewRegistry 创建新的 Registry 实例
func NewRegistry() Registry {
    return &registry{
        skills:  make(map[string]Skill),
        enabled: make(map[string]bool),
    }
}

// Register 实现
func (r *registry) Register(skill Skill) error {
    if skill == nil {
        return fmt.Errorf("skill cannot be nil")
    }
    
    name := skill.Name()
    if name == "" {
        return fmt.Errorf("skill name cannot be empty")
    }
    
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.skills[name]; exists {
        return fmt.Errorf("skill %q already registered", name)
    }
    
    r.skills[name] = skill
    r.enabled[name] = true  // 默认启用
    
    return nil
}

// Unregister 实现
func (r *registry) Unregister(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.skills[name]; !exists {
        return ErrSkillNotFound
    }
    
    delete(r.skills, name)
    delete(r.enabled, name)
    
    return nil
}

// Enable 实现
func (r *registry) Enable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.skills[name]; !exists {
        return ErrSkillNotFound
    }
    
    r.enabled[name] = true
    return nil
}

// Disable 实现
func (r *registry) Disable(name string) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.skills[name]; !exists {
        return ErrSkillNotFound
    }
    
    r.enabled[name] = false
    return nil
}

// Get 实现
func (r *registry) Get(name string) (Skill, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    skill, exists := r.skills[name]
    if !exists {
        return nil, ErrSkillNotFound
    }
    
    return skill, nil
}

// List 实现
func (r *registry) List() []Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    result := make([]Skill, 0, len(r.skills))
    for _, skill := range r.skills {
        result = append(result, skill)
    }
    
    return result
}

// ListEnabled 实现
func (r *registry) ListEnabled() []Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    result := make([]Skill, 0)
    for name, skill := range r.skills {
        if r.enabled[name] {
            result = append(result, skill)
        }
    }
    
    return result
}

// ToTools 实现
func (r *registry) ToTools() []llm.Tool {
    enabled := r.ListEnabled()
    tools := make([]llm.Tool, 0, len(enabled))
    
    for _, skill := range enabled {
        tool := llm.Tool{
            Type: "function",
            Function: llm.ToolFunction{
                Name:        skill.Name(),
                Description: skill.Description(),
                Parameters:  skill.Parameters(),
            },
        }
        tools = append(tools, tool)
    }
    
    return tools
}

// 错误定义
var (
    ErrSkillNotFound    = fmt.Errorf("skill not found")
    ErrSkillDisabled    = fmt.Errorf("skill is disabled")
    ErrInvalidConfig    = fmt.Errorf("invalid skill config")
    ErrProviderNotFound = fmt.Errorf("provider not found")
)
```

### 3.3 Provider 接口

```go
package skills

// Provider Skill 提供者接口
type Provider interface {
    // Type 返回 Provider 类型标识
    Type() string
    
    // Create 根据配置创建 Skill 实例
    Create(config SkillConfig) (Skill, error)
}

// ProviderRegistry Provider 注册中心
type ProviderRegistry struct {
    providers map[string]Provider
    mu        sync.RWMutex
}

// NewProviderRegistry 创建 Provider 注册中心
func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        providers: make(map[string]Provider),
    }
}

// Register 注册 Provider
func (pr *ProviderRegistry) Register(provider Provider) {
    pr.mu.Lock()
    defer pr.mu.Unlock()
    pr.providers[provider.Type()] = provider
}

// Get 获取 Provider
func (pr *ProviderRegistry) Get(providerType string) (Provider, error) {
    pr.mu.RLock()
    defer pr.mu.RUnlock()
    
    provider, exists := pr.providers[providerType]
    if !exists {
        return nil, ErrProviderNotFound
    }
    
    return provider, nil
}
```

### 3.4 配置结构

```go
package skills

// SkillConfig Skill 配置文件结构
type SkillConfig struct {
    // 基础信息
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description" json:"description"`
    
    // Provider 配置
    Provider string `yaml:"provider" json:"provider"` // builtin / http
    
    // HTTP Provider 特有配置
    Endpoint string            `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
    Timeout  int               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
    Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
    
    // 安全相关
    RiskLevel string `yaml:"risk_level,omitempty" json:"risk_level,omitempty"` // read / write / destructive
    
    // 参数定义
    Parameters ParameterSchema `yaml:"parameters" json:"parameters"`
}

// Validate 校验配置是否有效
func (c *SkillConfig) Validate() error {
    if c.Name == "" {
        return fmt.Errorf("skill name is required")
    }
    
    if c.Description == "" {
        return fmt.Errorf("skill description is required")
    }
    
    if c.Provider == "" {
        return fmt.Errorf("skill provider is required")
    }
    
    if c.Provider == "http" && c.Endpoint == "" {
        return fmt.Errorf("endpoint is required for http provider")
    }
    
    // 校验 RiskLevel
    if c.RiskLevel != "" && c.RiskLevel != "read" && c.RiskLevel != "write" && c.RiskLevel != "destructive" {
        return fmt.Errorf("invalid risk_level: %s", c.RiskLevel)
    }
    
    return nil
}
```

---

## 4. Provider 实现

### 4.1 Builtin Provider

```go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/weibh/openDeepWiki/backend/internal/pkg/skills"
)

// Provider 内置 Provider
type Provider struct {
    creators map[string]SkillCreator
}

// SkillCreator Skill 创建函数
type SkillCreator func(config skills.SkillConfig) (skills.Skill, error)

// NewProvider 创建 Builtin Provider
func NewProvider() *Provider {
    return &Provider{
        creators: make(map[string]SkillCreator),
    }
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
    return "builtin"
}

// Register 注册 Skill 创建器
func (p *Provider) Register(name string, creator SkillCreator) {
    p.creators[name] = creator
}

// Create 创建 Skill
func (p *Provider) Create(config skills.SkillConfig) (skills.Skill, error) {
    creator, exists := p.creators[config.Name]
    if !exists {
        return nil, fmt.Errorf("builtin skill %q not found", config.Name)
    }
    
    return creator(config)
}

// BuiltinSkill 内置 Skill 基类
type BuiltinSkill struct {
    config skills.SkillConfig
    fn     func(ctx context.Context, args json.RawMessage) (interface{}, error)
}

// Name 返回名称
func (s *BuiltinSkill) Name() string {
    return s.config.Name
}

// Description 返回描述
func (s *BuiltinSkill) Description() string {
    return s.config.Description
}

// Parameters 返回参数定义
func (s *BuiltinSkill) Parameters() skills.ParameterSchema {
    return s.config.Parameters
}

// Execute 执行
func (s *BuiltinSkill) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
    return s.fn(ctx, args)
}

// ProviderType 返回 Provider 类型
func (s *BuiltinSkill) ProviderType() string {
    return "builtin"
}

// NewBuiltinSkill 创建内置 Skill
func NewBuiltinSkill(config skills.SkillConfig, fn func(ctx context.Context, args json.RawMessage) (interface{}, error)) skills.Skill {
    return &BuiltinSkill{
        config: config,
        fn:     fn,
    }
}
```

### 4.2 HTTP Provider

```go
package http

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/weibh/openDeepWiki/backend/internal/pkg/skills"
)

// Provider HTTP Provider
type Provider struct {
    client *http.Client
}

// NewProvider 创建 HTTP Provider
func NewProvider() *Provider {
    return &Provider{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
    return "http"
}

// Create 创建 HTTP Skill
func (p *Provider) Create(config skills.SkillConfig) (skills.Skill, error) {
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    timeout := config.Timeout
    if timeout <= 0 {
        timeout = 30
    }
    
    return &HTTPSkill{
        config:  config,
        client:  p.client,
        timeout: time.Duration(timeout) * time.Second,
    }, nil
}

// HTTPSkill HTTP Skill 实现
type HTTPSkill struct {
    config  skills.SkillConfig
    client  *http.Client
    timeout time.Duration
}

// Name 返回名称
func (s *HTTPSkill) Name() string {
    return s.config.Name
}

// Description 返回描述
func (s *HTTPSkill) Description() string {
    return s.config.Description
}

// Parameters 返回参数定义
func (s *HTTPSkill) Parameters() skills.ParameterSchema {
    return s.config.Parameters
}

// Execute 执行 HTTP 调用
func (s *HTTPSkill) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
    // 创建带超时的上下文
    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()
    
    // 创建请求
    req, err := http.NewRequestWithContext(ctx, "POST", s.config.Endpoint, bytes.NewReader(args))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    // 设置 Headers
    req.Header.Set("Content-Type", "application/json")
    for key, value := range s.config.Headers {
        req.Header.Set(key, value)
    }
    
    // 发送请求
    resp, err := s.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // 解析响应
    var result interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return result, nil
}

// ProviderType 返回 Provider 类型
func (s *HTTPSkill) ProviderType() string {
    return "http"
}
```

---

## 5. 配置加载器设计

### 5.1 目录解析

```go
package skills

import (
    "fmt"
    "os"
    "path/filepath"
)

// ResolveSkillsDir 解析 Skills 目录
// 优先级：环境变量 > 配置文件 > 默认目录
func ResolveSkillsDir(configDir string) (string, error) {
    // 1. 检查环境变量
    if dir := os.Getenv("SKILLS_DIR"); dir != "" {
        return filepath.Abs(dir)
    }
    
    // 2. 检查配置文件
    if configDir != "" {
        return filepath.Abs(configDir)
    }
    
    // 3. 默认目录
    exePath, err := os.Executable()
    if err != nil {
        return "", err
    }
    
    defaultDir := filepath.Join(filepath.Dir(exePath), "skills")
    return defaultDir, nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
    info, err := os.Stat(dir)
    if os.IsNotExist(err) {
        return os.MkdirAll(dir, 0755)
    }
    if err != nil {
        return err
    }
    if !info.IsDir() {
        return fmt.Errorf("%s is not a directory", dir)
    }
    return nil
}
```

### 5.2 文件监听器

```go
package skills

import (
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// FileWatcher 文件监听器
type FileWatcher struct {
    dir      string
    interval time.Duration
    stop     chan struct{}
    callback func(event FileEvent)
    files    map[string]os.FileInfo
}

// FileEvent 文件事件
type FileEvent struct {
    Type string // create, modify, delete
    Path string
    Info os.FileInfo
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher(dir string, interval time.Duration, callback func(event FileEvent)) *FileWatcher {
    return &FileWatcher{
        dir:      dir,
        interval: interval,
        stop:     make(chan struct{}),
        callback: callback,
        files:    make(map[string]os.FileInfo),
    }
}

// Start 启动监听
func (w *FileWatcher) Start() {
    // 初始扫描
    w.scan()
    
    // 定时扫描
    ticker := time.NewTicker(w.interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                w.scan()
            case <-w.stop:
                ticker.Stop()
                return
            }
        }
    }()
}

// Stop 停止监听
func (w *FileWatcher) Stop() {
    close(w.stop)
}

// scan 扫描目录变化
func (w *FileWatcher) scan() {
    currentFiles := make(map[string]os.FileInfo)
    
    err := filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        // 只处理 YAML/JSON 文件
        if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".json")) {
            currentFiles[path] = info
        }
        
        return nil
    })
    
    if err != nil {
        log.Printf("Failed to scan skills directory: %v", err)
        return
    }
    
    // 检测新增和修改
    for path, info := range currentFiles {
        oldInfo, exists := w.files[path]
        if !exists {
            w.callback(FileEvent{Type: "create", Path: path, Info: info})
        } else if info.ModTime() != oldInfo.ModTime() || info.Size() != oldInfo.Size() {
            w.callback(FileEvent{Type: "modify", Path: path, Info: info})
        }
    }
    
    // 检测删除
    for path, info := range w.files {
        if _, exists := currentFiles[path]; !exists {
            w.callback(FileEvent{Type: "delete", Path: path, Info: info})
        }
    }
    
    w.files = currentFiles
}
```

### 5.3 配置加载器

```go
package skills

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
    registry  Registry
    providers *ProviderRegistry
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(registry Registry, providers *ProviderRegistry) *ConfigLoader {
    return &ConfigLoader{
        registry:  registry,
        providers: providers,
    }
}

// LoadFromDir 从目录加载所有配置
func (l *ConfigLoader) LoadFromDir(dir string) error {
    files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
    if err != nil {
        return err
    }
    
    ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
    if err != nil {
        return err
    }
    files = append(files, ymlFiles...)
    
    jsonFiles, err := filepath.Glob(filepath.Join(dir, "*.json"))
    if err != nil {
        return err
    }
    files = append(files, jsonFiles...)
    
    for _, file := range files {
        if err := l.LoadFromFile(file); err != nil {
            log.Printf("Failed to load skill config from %s: %v", file, err)
            continue
        }
    }
    
    return nil
}

// LoadFromFile 从文件加载配置
func (l *ConfigLoader) LoadFromFile(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }
    
    var config SkillConfig
    
    // 根据扩展名解析
    ext := filepath.Ext(path)
    switch ext {
    case ".yaml", ".yml":
        err = yaml.Unmarshal(data, &config)
    case ".json":
        err = json.Unmarshal(data, &config)
    default:
        return fmt.Errorf("unsupported file format: %s", ext)
    }
    
    if err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }
    
    // 校验配置
    if err := config.Validate(); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }
    
    // 获取 Provider
    provider, err := l.providers.Get(config.Provider)
    if err != nil {
        return fmt.Errorf("provider not found: %w", err)
    }
    
    // 创建 Skill
    skill, err := provider.Create(config)
    if err != nil {
        return fmt.Errorf("failed to create skill: %w", err)
    }
    
    // 注册到 Registry
    // 如果已存在，先注销
    if _, err := l.registry.Get(config.Name); err == nil {
        l.registry.Unregister(config.Name)
    }
    
    if err := l.registry.Register(skill); err != nil {
        return fmt.Errorf("failed to register skill: %w", err)
    }
    
    return nil
}

// UnloadFromFile 根据文件路径卸载 Skill
func (l *ConfigLoader) UnloadFromFile(path string) error {
    // 从文件名推断 Skill 名称
    basename := filepath.Base(path)
    ext := filepath.Ext(basename)
    name := basename[:len(basename)-len(ext)]
    
    // 尝试从 Registry 获取
    skill, err := l.registry.Get(name)
    if err != nil {
        // 可能名称不匹配，尝试从文件内容读取
        data, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        
        var config SkillConfig
        if err := yaml.Unmarshal(data, &config); err != nil {
            return err
        }
        name = config.Name
    } else {
        name = skill.Name()
    }
    
    return l.registry.Unregister(name)
}
```

---

## 6. 初始化流程

```go
package skills

import (
    "log"
    "time"
    
    "github.com/weibh/openDeepWiki/backend/internal/pkg/skills/builtin"
    httpprovider "github.com/weibh/openDeepWiki/backend/internal/pkg/skills/http"
)

// Manager Skills 管理器
type Manager struct {
    Registry  Registry
    Loader    *ConfigLoader
    Watcher   *FileWatcher
    providers *ProviderRegistry
}

// NewManager 创建 Skills 管理器
func NewManager(configDir string) (*Manager, error) {
    // 解析目录
    skillsDir, err := ResolveSkillsDir(configDir)
    if err != nil {
        return nil, err
    }
    
    // 确保目录存在
    if err := EnsureDir(skillsDir); err != nil {
        return nil, err
    }
    
    // 创建 Registry
    registry := NewRegistry()
    
    // 创建 Provider 注册中心
    providers := NewProviderRegistry()
    
    // 注册 Builtin Provider
    builtinProvider := builtin.NewProvider()
    providers.Register(builtinProvider)
    
    // 注册 HTTP Provider
    httpProvider := httpprovider.NewProvider()
    providers.Register(httpProvider)
    
    // 创建配置加载器
    loader := NewConfigLoader(registry, providers)
    
    // 初始加载
    if err := loader.LoadFromDir(skillsDir); err != nil {
        log.Printf("Failed to load skills from %s: %v", skillsDir, err)
    }
    
    // 创建文件监听器
    watcher := NewFileWatcher(skillsDir, 5*time.Second, func(event FileEvent) {
        switch event.Type {
        case "create", "modify":
            if err := loader.LoadFromFile(event.Path); err != nil {
                log.Printf("Failed to reload skill from %s: %v", event.Path, err)
            } else {
                log.Printf("Reloaded skill from %s", event.Path)
            }
        case "delete":
            if err := loader.UnloadFromFile(event.Path); err != nil {
                log.Printf("Failed to unload skill from %s: %v", event.Path, err)
            } else {
                log.Printf("Unloaded skill from %s", event.Path)
            }
        }
    })
    
    // 启动监听
    watcher.Start()
    
    return &Manager{
        Registry:  registry,
        Loader:    loader,
        Watcher:   watcher,
        providers: providers,
    }, nil
}

// Stop 停止管理器
func (m *Manager) Stop() {
    if m.Watcher != nil {
        m.Watcher.Stop()
    }
}

// RegisterBuiltin 注册内置 Skill
func (m *Manager) RegisterBuiltin(name string, creator builtin.SkillCreator) {
    if p, ok := m.providers.Get("builtin"); ok == nil {
        if bp, ok := p.(*builtin.Provider); ok {
            bp.Register(name, creator)
        }
    }
}
```

---

## 7. 与 LLM Client 集成

```go
package llm

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/weibh/openDeepWiki/backend/internal/pkg/skills"
)

// SkillExecutor Skill 执行器
type SkillExecutor struct {
    registry skills.Registry
}

// NewSkillExecutor 创建 Skill 执行器
func NewSkillExecutor(registry skills.Registry) *SkillExecutor {
    return &SkillExecutor{registry: registry}
}

// Execute 执行 Tool Call
func (e *SkillExecutor) Execute(ctx context.Context, toolCall ToolCall) (ToolResult, error) {
    // 从 Registry 获取 Skill
    skill, err := e.registry.Get(toolCall.Function.Name)
    if err != nil {
        return ToolResult{
            Content: fmt.Sprintf("Skill not found: %s", toolCall.Function.Name),
            IsError: true,
        }, nil
    }
    
    // 执行 Skill
    result, err := skill.Execute(ctx, json.RawMessage(toolCall.Function.Arguments))
    if err != nil {
        return ToolResult{
            Content: fmt.Sprintf("Execution failed: %v", err),
            IsError: true,
        }, nil
    }
    
    // 序列化结果
    content, err := json.Marshal(result)
    if err != nil {
        return ToolResult{
            Content: fmt.Sprintf("Failed to serialize result: %v", err),
            IsError: true,
        }, nil
    }
    
    return ToolResult{
        Content: string(content),
        IsError: false,
    }, nil
}
```

---

## 8. 代码目录结构

```
backend/internal/pkg/skills/
├── skill.go              # Skill 接口定义
├── registry.go           # Registry 接口与实现
├── provider.go           # Provider 接口
├── config.go             # 配置结构定义
├── loader.go             # 配置加载
├── watcher.go            # 文件监听
├── manager.go            # 管理器（初始化入口）
├── errors.go             # 错误定义
├── builtin/
│   ├── provider.go       # Builtin Provider
│   └── skills/           # 内置 Skills 实现
│       └── example.go    # 示例 Skill
├── http/
│   ├── provider.go       # HTTP Provider
│   └── skill.go          # HTTP Skill 实现
└── README.md             # 使用文档

# Skills 配置目录（运行时）
skills/
└── example.yaml          # 示例配置
```

---

## 9. 使用示例

### 9.1 初始化 Skills

```go
// main.go
import "github.com/weibh/openDeepWiki/backend/internal/pkg/skills"

func main() {
    // 创建 Skills 管理器
    skillsManager, err := skills.NewManager(cfg.Skills.Dir)
    if err != nil {
        log.Fatal(err)
    }
    defer skillsManager.Stop()
    
    // 获取 Registry
    registry := skillsManager.Registry
    
    // ... 其他初始化
}
```

### 9.2 LLM Client 使用 Skills

```go
// 创建 Skill 执行器
skillExecutor := llm.NewSkillExecutor(registry)

// 获取 Tools
tools := registry.ToTools()

// 执行对话
messages := []llm.ChatMessage{
    {Role: "system", Content: "你是一个代码分析助手..."},
    {Role: "user", Content: "分析这个项目的架构"},
}

response, err := client.ChatWithToolExecution(ctx, messages, tools, skillExecutor)
```

### 9.3 添加内置 Skill

```go
// 注册内置 Skill
skillsManager.RegisterBuiltin("count_go_files", func(config skills.SkillConfig) (skills.Skill, error) {
    return builtin.NewBuiltinSkill(config, func(ctx context.Context, args json.RawMessage) (interface{}, error) {
        // 实现逻辑
        return map[string]int{"count": 42}, nil
    }), nil
})
```

### 9.4 Skill 配置文件示例

```yaml
# skills/search_logs.yaml
name: search_logs
description: 搜索 Kubernetes Pod 日志，返回匹配的日志行
provider: http
endpoint: http://127.0.0.1:8081/execute
timeout: 30
headers:
  Authorization: Bearer ${K8S_TOKEN}
risk_level: read
parameters:
  type: object
  properties:
    namespace:
      type: string
      description: Kubernetes 命名空间
    pod:
      type: string
      description: Pod 名称，支持通配符
    keyword:
      type: string
      description: 搜索关键词
    limit:
      type: integer
      description: 返回最大行数
      default: 100
  required:
    - namespace
    - pod
```

---

## 10. 测试策略

### 10.1 单元测试

| 模块 | 测试内容 |
|------|---------|
| Registry | 并发注册/注销、启用/禁用、状态一致性 |
| Provider | Builtin 和 HTTP Provider 创建 Skill |
| Loader | 配置文件解析、错误处理 |
| Watcher | 文件事件检测 |

### 10.2 集成测试

- 完整初始化流程
- 文件变更热加载
- LLM Client 集成

---

## 11. 实现计划

| 阶段 | 内容 | 预估时间 |
|------|------|---------|
| 1 | 核心接口定义（Skill、Registry、Provider） | 1h |
| 2 | Registry 实现 | 2h |
| 3 | HTTP Provider 实现 | 2h |
| 4 | 配置加载与文件监听 | 2h |
| 5 | Manager 和初始化流程 | 1h |
| 6 | LLM Client 集成 | 1h |
| 7 | 单元测试 | 2h |
| 8 | 示例 Skill 和文档 | 1h |
| **总计** | | **12h** |
