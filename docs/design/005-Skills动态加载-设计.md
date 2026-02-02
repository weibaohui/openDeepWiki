# 005-Skills动态加载-设计.md

## 1. 架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         LLM Client                          │
│  ┌───────────────────────────────────────────────────────┐ │
│  │              Chat / GenerateDocument                  │ │
│  └───────────────────────────────────────────────────────┘ │
│                           │                                 │
│                           ▼                                 │
│  ┌───────────────────────────────────────────────────────┐ │
│  │              Skill Injector                           │ │
│  │  将匹配的 Skills 注入 System Prompt                   │ │
│  └───────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                   Skill Matcher                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  - 分析任务描述                                       │ │
│  │  - 匹配 Skills description 关键字                     │ │
│  │  - 计算匹配分数                                       │ │
│  │  - 返回排序后的匹配列表                               │ │
│  └───────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                   Skill Registry                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   skills    │  │   status    │  │      metadata       │ │
│  │  (map[name] │  │  (map[name] │  │   index for search  │ │
│  │   *Skill)   │  │   bool)     │  │                     │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ Skill Loader  │   │ Skill Parser  │   │ File Watcher  │
│               │   │               │   │               │
│ 目录扫描       │   │ YAML 解析      │   │ 热加载监听     │
│ 资源发现       │   │ Markdown 提取  │   │ 变更检测       │
└───────────────┘   └───────────────┘   └───────────────┘
        │
        ▼
┌───────────────┐
│  skills/      │
│  ├── skill-a/ │
│  │   └── SKILL.md
│  ├── skill-b/ │
│  │   ├── SKILL.md
│  │   └── references/
│  └── ...      │
└───────────────┘
```

### 1.2 数据流

```
1. 系统启动
   │
   ├── 解析 Skills 目录
   │
   ├── 遍历每个子目录
   │   ├── 读取 SKILL.md
   │   ├── 解析 YAML frontmatter（元数据）
   │   ├── 提取 Markdown body（指令）
   │   ├── 检查 scripts/ references/ assets/
   │   └── 创建 Skill 对象（仅保存元数据，body 按需加载）
   │
   └── 注册到 Registry（仅元数据索引）

2. 任务执行
   │
   ├── 接收任务（类型、描述、上下文）
   │
   ├── Matcher 分析任务
   │   ├── 提取关键词
   │   ├── 与 Skills description 匹配
   │   └── 计算匹配分数
   │
   ├── 加载匹配 Skills 的完整指令
   │   ├── 读取 SKILL.md body
   │   └── 加载所需的 references/
   │
   ├── Injector 构建 Skill 上下文
   │   └── 格式化 Skills 内容
   │
   └── 注入到 LLM System Prompt
       └── 执行对话
```

---

## 2. 核心组件设计

### 2.1 Skill 结构

```go
package skills

import (
    "time"
)

// Skill 技能定义
type Skill struct {
    // 元数据（始终加载）
    Name          string            `yaml:"name" json:"name"`
    Description   string            `yaml:"description" json:"description"`
    License       string            `yaml:"license,omitempty" json:"license,omitempty"`
    Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
    Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
    AllowedTools  string            `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`

    // 路径信息
    Path          string `json:"path"`          // Skill 目录绝对路径
    SkillMDPath   string `json:"skill_md_path"` // SKILL.md 文件路径

    // 资源标志（用于判断是否存在子目录）
    HasScripts    bool `json:"has_scripts"`
    HasReferences bool `json:"has_references"`
    HasAssets     bool `json:"has_assets"`

    // 状态
    Enabled       bool      `json:"enabled"`
    LoadedAt      time.Time `json:"loaded_at"`
}

// SkillContent Skill 完整内容（按需加载）
type SkillContent struct {
    Skill        *Skill
    Instructions string            // SKILL.md body (frontmatter 之后的内容)
    References   map[string]string // references/ 下的文件内容
}
```

### 2.2 Skill Parser

```go
package skills

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/weibaohui/opendeepwiki/backend/internal/pkg/llm"
    "gopkg.in/yaml.v3"
)

// Parser Skill 解析器
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

// Parse 完整解析 Skill 目录
func (p *Parser) Parse(skillPath string) (*Skill, string, error) {
    skillPath = filepath.Clean(skillPath)
    skillMDPath := filepath.Join(skillPath, "SKILL.md")

    // 检查 SKILL.md 是否存在
    if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
        return nil, "", fmt.Errorf("SKILL.md not found in %s", skillPath)
    }

    // 读取文件内容
    content, err := os.ReadFile(skillMDPath)
    if err != nil {
        return nil, "", fmt.Errorf("failed to read SKILL.md: %w", err)
    }

    // 解析 frontmatter 和 body
    skill, body, err := p.parseSkillMD(string(content))
    if err != nil {
        return nil, "", err
    }

    // 设置路径信息
    skill.Path = skillPath
    skill.SkillMDPath = skillMDPath
    skill.LoadedAt = time.Now()

    // 检查资源目录
    skill.HasScripts = p.dirExists(filepath.Join(skillPath, "scripts"))
    skill.HasReferences = p.dirExists(filepath.Join(skillPath, "references"))
    skill.HasAssets = p.dirExists(filepath.Join(skillPath, "assets"))

    // 校验
    if err := p.Validate(skill); err != nil {
        return nil, "", err
    }

    return skill, body, nil
}

// ParseMetadata 仅解析元数据（快速）
func (p *Parser) ParseMetadata(skillPath string) (*Skill, error) {
    skillPath = filepath.Clean(skillPath)
    skillMDPath := filepath.Join(skillPath, "SKILL.md")

    content, err := os.ReadFile(skillMDPath)
    if err != nil {
        return nil, err
    }

    skill, _, err := p.parseSkillMD(string(content))
    if err != nil {
        return nil, err
    }

    skill.Path = skillPath
    skill.SkillMDPath = skillMDPath
    skill.LoadedAt = time.Now()

    return skill, nil
}

// parseSkillMD 解析 SKILL.md 内容
func (p *Parser) parseSkillMD(content string) (*Skill, string, error) {
    skill := &Skill{}

    // 检查 frontmatter
    if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
        return nil, "", fmt.Errorf("SKILL.md must start with YAML frontmatter")
    }

    // 找到 frontmatter 结束位置
    endIdx := strings.Index(content[3:], "\n---")
    if endIdx == -1 {
        return nil, "", fmt.Errorf("YAML frontmatter not properly closed")
    }
    endIdx += 3 // 加上前面的 "---"

    // 提取 YAML
    yamlContent := content[3:endIdx]
    body := strings.TrimSpace(content[endIdx+4:])

    // 解析 YAML
    if err := yaml.Unmarshal([]byte(yamlContent), skill); err != nil {
        return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
    }

    return skill, body, nil
}

// Validate 校验 Skill
func (p *Parser) Validate(skill *Skill) error {
    // 校验 name
    if skill.Name == "" {
        return fmt.Errorf("name is required")
    }
    if len(skill.Name) > p.maxNameLen {
        return fmt.Errorf("name exceeds %d characters", p.maxNameLen)
    }
    if !isValidSkillName(skill.Name) {
        return fmt.Errorf("name must contain only lowercase letters, numbers, and hyphens")
    }

    // 校验 description
    if skill.Description == "" {
        return fmt.Errorf("description is required")
    }
    if len(skill.Description) > p.maxDescriptionLen {
        return fmt.Errorf("description exceeds %d characters", p.maxDescriptionLen)
    }

    return nil
}

// dirExists 检查目录是否存在
func (p *Parser) dirExists(path string) bool {
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    return info.IsDir()
}

// isValidSkillName 校验 name 格式
func isValidSkillName(name string) bool {
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
    for _, c := range name {
        if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
            return false
        }
    }
    return true
}
```

### 2.3 Skill Loader

```go
package skills

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// Loader Skill 加载器
type Loader struct {
    parser    *Parser
    registry  Registry
    mu        sync.RWMutex
    skillBodies map[string]string // name -> body (缓存)
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry Registry) *Loader {
    return &Loader{
        parser:      parser,
        registry:    registry,
        skillBodies: make(map[string]string),
    }
}

// LoadFromDir 从目录加载所有 Skills
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error) {
    dir = filepath.Clean(dir)
    
    // 检查目录是否存在
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        log.Printf("Skills directory does not exist: %s", dir)
        return nil, nil
    }

    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, fmt.Errorf("failed to read skills directory: %w", err)
    }

    results := make([]*LoadResult, 0)

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        skillPath := filepath.Join(dir, entry.Name())
        result := l.loadSkill(skillPath)
        results = append(results, result)
    }

    return results, nil
}

// LoadFromPath 加载单个 Skill
func (l *Loader) LoadFromPath(path string) (*Skill, error) {
    result := l.loadSkill(path)
    if result.Error != nil {
        return nil, result.Error
    }
    return result.Skill, nil
}

// loadSkill 加载 Skill（内部）
func (l *Loader) loadSkill(path string) *LoadResult {
    skill, body, err := l.parser.Parse(path)
    if err != nil {
        return &LoadResult{
            Error:  err,
            Action: "failed",
        }
    }

    // 缓存 body
    l.mu.Lock()
    l.skillBodies[skill.Name] = body
    l.mu.Unlock()

    // 检查是否已存在
    existing, _ := l.registry.Get(skill.Name)
    action := "created"
    if existing != nil {
        action = "updated"
    }

    // 注册到 Registry
    if err := l.registry.Register(skill); err != nil {
        return &LoadResult{
            Skill:  skill,
            Error:  err,
            Action: "failed",
        }
    }

    return &LoadResult{
        Skill:  skill,
        Action: action,
    }
}

// GetBody 获取 Skill 的 body 内容
func (l *Loader) GetBody(name string) (string, error) {
    l.mu.RLock()
    body, exists := l.skillBodies[name]
    l.mu.RUnlock()

    if exists {
        return body, nil
    }

    // 如果缓存中没有，尝试从文件加载
    skill, err := l.registry.Get(name)
    if err != nil {
        return "", err
    }

    // 重新解析获取 body
    _, body, err = l.parser.Parse(skill.Path)
    if err != nil {
        return "", err
    }

    // 缓存
    l.mu.Lock()
    l.skillBodies[name] = body
    l.mu.Unlock()

    return body, nil
}

// LoadReferences 加载 references/ 下的文件
func (l *Loader) LoadReferences(skill *Skill) (map[string]string, error) {
    if !skill.HasReferences {
        return nil, nil
    }

    refsDir := filepath.Join(skill.Path, "references")
    entries, err := os.ReadDir(refsDir)
    if err != nil {
        return nil, err
    }

    refs := make(map[string]string)
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        path := filepath.Join(refsDir, entry.Name())
        content, err := os.ReadFile(path)
        if err != nil {
            log.Printf("Failed to read reference file %s: %v", path, err)
            continue
        }
        refs[entry.Name()] = string(content)
    }

    return refs, nil
}

// Unload 卸载 Skill
func (l *Loader) Unload(name string) error {
    l.mu.Lock()
    delete(l.skillBodies, name)
    l.mu.Unlock()

    return l.registry.Unregister(name)
}

// Reload 重新加载 Skill
func (l *Loader) Reload(name string) (*Skill, error) {
    skill, err := l.registry.Get(name)
    if err != nil {
        return nil, err
    }

    l.Unload(name)
    return l.LoadFromPath(skill.Path)
}

// LoadResult 加载结果
type LoadResult struct {
    Skill  *Skill
    Error  error
    Action string // "created", "updated", "failed"
}
```

### 2.4 Skill Matcher

```go
package skills

import (
    "strings"
)

// Task 任务定义
type Task struct {
    Type        string   // 任务类型（overview, architecture, api等）
    Description string   // 任务描述
    RepoType    string   // 仓库类型（go, python, java等）
    Tags        []string // 标签
}

// Match 匹配结果
type Match struct {
    Skill  *Skill
    Score  float64 // 匹配分数 0-1
    Reason string  // 匹配原因
}

// Matcher Skill 匹配器
type Matcher struct {
    registry Registry
}

// NewMatcher 创建匹配器
func NewMatcher(registry Registry) *Matcher {
    return &Matcher{registry: registry}
}

// Match 根据任务匹配 Skills
func (m *Matcher) Match(task Task) ([]*Match, error) {
    enabled := m.registry.ListEnabled()
    matches := make([]*Match, 0)

    for _, skill := range enabled {
        score, reason := m.calculateScore(skill, task)
        if score > 0 {
            matches = append(matches, &Match{
                Skill:  skill,
                Score:  score,
                Reason: reason,
            })
        }
    }

    // 按分数排序
    sortMatches(matches)
    return matches, nil
}

// MatchByDescription 根据描述匹配（简单版本）
func (m *Matcher) MatchByDescription(description string) ([]*Match, error) {
    task := Task{Description: description}
    return m.Match(task)
}

// calculateScore 计算匹配分数
func (m *Matcher) calculateScore(skill *Skill, task Task) (float64, string) {
    score := 0.0
    reasons := make([]string, 0)

    desc := strings.ToLower(skill.Description)
    taskDesc := strings.ToLower(task.Description)
    taskType := strings.ToLower(task.Type)
    repoType := strings.ToLower(task.RepoType)

    // 1. 描述关键词匹配（最高权重）
    keywords := extractKeywords(taskDesc)
    keywordMatches := 0
    for _, kw := range keywords {
        if strings.Contains(desc, kw) {
            keywordMatches++
        }
    }
    if len(keywords) > 0 {
        matchRatio := float64(keywordMatches) / float64(len(keywords))
        score += matchRatio * 0.5
        if matchRatio > 0.5 {
            reasons = append(reasons, "keyword match")
        }
    }

    // 2. 任务类型匹配
    if taskType != "" && strings.Contains(desc, taskType) {
        score += 0.3
        reasons = append(reasons, "task type match")
    }

    // 3. 仓库类型匹配
    if repoType != "" && strings.Contains(desc, repoType) {
        score += 0.2
        reasons = append(reasons, "repo type match")
    }

    // 4. 标签匹配
    for _, tag := range task.Tags {
        if strings.Contains(desc, strings.ToLower(tag)) {
            score += 0.1
            reasons = append(reasons, "tag match")
            break
        }
    }

    if score == 0 {
        return 0, ""
    }

    return score, strings.Join(reasons, ", ")
}

// extractKeywords 提取关键词
func extractKeywords(text string) []string {
    // 简单的关键词提取：过滤常见停用词，保留长度>2的词
    stopWords := map[string]bool{
        "the": true, "a": true, "an": true, "and": true, "or": true,
        "is": true, "are": true, "was": true, "were": true,
        "this": true, "that": true, "these": true, "those": true,
        "to": true, "of": true, "in": true, "for": true, "on": true,
        "with": true, "by": true, "at": true, "from": true,
    }

    words := strings.Fields(text)
    keywords := make([]string, 0)
    seen := make(map[string]bool)

    for _, word := range words {
        word = strings.Trim(word, ".,!?;:()[]{}\"")
        word = strings.ToLower(word)
        if len(word) > 2 && !stopWords[word] && !seen[word] {
            keywords = append(keywords, word)
            seen[word] = true
        }
    }

    return keywords
}

// sortMatches 排序匹配结果（按分数降序）
func sortMatches(matches []*Match) {
    // 简单冒泡排序（数据量小）
    for i := 0; i < len(matches); i++ {
        for j := i + 1; j < len(matches); j++ {
            if matches[j].Score > matches[i].Score {
                matches[i], matches[j] = matches[j], matches[i]
            }
        }
    }
}
```

### 2.5 Skill Injector

```go
package skills

import (
    "fmt"
    "strings"
)

// Injector Skill 注入器
type Injector struct {
    loader *Loader
}

// NewInjector 创建注入器
func NewInjector(loader *Loader) *Injector {
    return &Injector{loader: loader}
}

// InjectToPrompt 将 Skills 注入到 System Prompt
func (i *Injector) InjectToPrompt(systemPrompt string, matches []*Match) (string, error) {
    if len(matches) == 0 {
        return systemPrompt, nil
    }

    // 构建 Skills 上下文
    skillContext, err := i.BuildSkillContext(matches)
    if err != nil {
        return "", err
    }

    // 注入到 Prompt
    var sb strings.Builder
    sb.WriteString(systemPrompt)
    sb.WriteString("\n\n")
    sb.WriteString(skillContext)

    return sb.String(), nil
}

// BuildSkillContext 构建 Skills 上下文
func (i *Injector) BuildSkillContext(matches []*Match) (string, error) {
    var sb strings.Builder

    sb.WriteString("## 专业技能指导\n\n")
    sb.WriteString("在完成以下任务时，请遵循相关技能的专业指导：\n\n")

    for idx, match := range matches {
        skill := match.Skill

        sb.WriteString(fmt.Sprintf("### 技能 %d: %s\n", idx+1, skill.Name))
        sb.WriteString(fmt.Sprintf("*匹配原因: %s (%.0f%%)*\n\n", match.Reason, match.Score*100))

        // 获取指令内容
        body, err := i.loader.GetBody(skill.Name)
        if err != nil {
            sb.WriteString(fmt.Sprintf("*(无法加载技能内容: %v)*\n", err))
            continue
        }

        sb.WriteString(body)
        sb.WriteString("\n\n---\n\n")
    }

    return sb.String(), nil
}

// BuildSingleSkillContext 构建单个 Skill 上下文
func (i *Injector) BuildSingleSkillContext(skill *Skill) (string, error) {
    body, err := i.loader.GetBody(skill.Name)
    if err != nil {
        return "", err
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("## 技能: %s\n\n", skill.Name))
    sb.WriteString(body)

    return sb.String(), nil
}
```

### 2.6 Manager（整合）

```go
package skills

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
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
    return &Config{
        Dir:            "./skills",
        AutoReload:     true,
        ReloadInterval: 5 * time.Second,
    }
}

// Manager Skills 管理器
type Manager struct {
    Config   *Config
    Registry Registry
    Parser   *Parser
    Loader   *Loader
    Matcher  *Matcher
    Injector *Injector
    watcher  *FileWatcher
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error) {
    if config == nil {
        config = DefaultConfig()
    }

    // 解析目录
    dir, err := resolveSkillsDir(config.Dir)
    if err != nil {
        return nil, err
    }
    config.Dir = dir

    log.Printf("Skills directory: %s", dir)

    // 确保目录存在
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create skills directory: %w", err)
    }

    // 创建组件
    registry := NewRegistry()
    parser := NewParser()
    loader := NewLoader(parser, registry)
    matcher := NewMatcher(registry)
    injector := NewInjector(loader)

    m := &Manager{
        Config:   config,
        Registry: registry,
        Parser:   parser,
        Loader:   loader,
        Matcher:  matcher,
        Injector: injector,
    }

    // 初始加载
    if _, err := loader.LoadFromDir(dir); err != nil {
        log.Printf("Warning: failed to load skills: %v", err)
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
            if isSkillDir(event.Path) {
                log.Printf("Loading new skill from %s", event.Path)
                if _, err := m.Loader.LoadFromPath(event.Path); err != nil {
                    log.Printf("Failed to load skill: %v", err)
                }
            }
        case "modify":
            // 检查是否是 SKILL.md 被修改
            if filepath.Base(event.Path) == "SKILL.md" {
                skillDir := filepath.Dir(event.Path)
                skillName := filepath.Base(skillDir)
                log.Printf("Reloading skill: %s", skillName)
                if _, err := m.Loader.Reload(skillName); err != nil {
                    log.Printf("Failed to reload skill: %v", err)
                }
            }
        case "delete":
            // 尝试根据路径推断 skill name
            skillName := filepath.Base(event.Path)
            if _, err := m.Registry.Get(skillName); err == nil {
                log.Printf("Unloading skill: %s", skillName)
                if err := m.Loader.Unload(skillName); err != nil {
                    log.Printf("Failed to unload skill: %v", err)
                }
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

// MatchAndInject 匹配 Skills 并注入 Prompt
func (m *Manager) MatchAndInject(systemPrompt string, task Task) (string, []*Match, error) {
    // 匹配 Skills
    matches, err := m.Matcher.Match(task)
    if err != nil {
        return "", nil, err
    }

    // 注入 Prompt
    newPrompt, err := m.Injector.InjectToPrompt(systemPrompt, matches)
    if err != nil {
        return "", nil, err
    }

    return newPrompt, matches, nil
}

// resolveSkillsDir 解析 Skills 目录
func resolveSkillsDir(configDir string) (string, error) {
    // 1. 环境变量
    if dir := os.Getenv("SKILLS_DIR"); dir != "" {
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
        return filepath.Join(cwd, "skills"), nil
    }
    return filepath.Join(filepath.Dir(exePath), "skills"), nil
}

// isSkillDir 检查是否是 Skill 目录
func isSkillDir(path string) bool {
    skillMD := filepath.Join(path, "SKILL.md")
    _, err := os.Stat(skillMD)
    return err == nil
}
```

---

## 3. 示例 Skill

```markdown
# skills/go-analysis/SKILL.md

---
name: go-analysis
description: Analyze Go projects to identify architecture patterns, module dependencies, and code organization. Use when working with Go repositories or when the user asks about Go project structure.
license: MIT
compatibility: Requires Go 1.18+
metadata:
  author: openDeepWiki
  version: "1.0"
---

# Go 项目分析指南

## 分析步骤

1. **识别项目结构**
   - 查找 `go.mod` 了解模块路径和依赖
   - 分析目录结构，识别主要包（pkg/, cmd/, internal/ 等）
   - 找出入口点（main 包）

2. **分析架构模式**
   - 检查是否使用分层架构（handler -> service -> repository）
   - 识别接口定义和实现分离
   - 查找依赖注入的使用

3. **核心组件识别**
   - 列出所有公开 API（HTTP handlers / gRPC services）
   - 识别业务逻辑层（services/usecases）
   - 找到数据访问层（repositories / DAOs）

4. **依赖关系**
   - 分析 import 关系
   - 识别循环依赖
   - 找出核心依赖库的作用

## 输出格式

生成以下文档：
- `overview.md`: 项目概述、技术栈
- `architecture.md`: 架构分析、模块划分
- `api.md`: API 接口文档
- `business-flow.md`: 业务流程

## 注意事项

- 关注 `internal/` 包的封装性
- 注意接口定义的抽象程度
- 检查错误处理模式
- 识别并发和通道的使用
```

---

## 4. 测试策略

### 4.1 Parser 测试

- [ ] 正常解析 SKILL.md
- [ ] 缺少 frontmatter
- [ ] name 格式错误
- [ ] description 过长
- [ ] YAML 语法错误

### 4.2 Loader 测试

- [ ] 加载目录下所有 Skills
- [ ] 加载单个 Skill
- [ ] 重新加载更新
- [ ] 卸载 Skill

### 4.3 Matcher 测试

- [ ] 关键词匹配
- [ ] 任务类型匹配
- [ ] 多 Skills 排序
- [ ] 无匹配情况

### 4.4 Injector 测试

- [ ] 构建 Skill 上下文
- [ ] 注入到 Prompt
- [ ] 多个 Skills 组合

### 4.5 集成测试

- [ ] 完整流程：加载 -> 匹配 -> 注入
- [ ] 热加载测试
- [ ] 并发安全测试

---

## 5. 代码目录结构

```
backend/internal/pkg/skills/
├── skill.go              # Skill 结构定义
├── registry.go           # Registry 接口与实现
├── parser.go             # SKILL.md 解析器
├── loader.go             # Skill 加载器
├── matcher.go            # Skill 匹配器
├── injector.go           # Prompt 注入器
├── manager.go            # 管理器（整合）
├── watcher.go            # 文件监听
├── errors.go             # 错误定义
├── types.go              # 公共类型
├── parser_test.go        # 解析器测试
├── loader_test.go        # 加载器测试
├── matcher_test.go       # 匹配器测试
└── README.md             # 使用文档

skills/                   # Skills 目录
└── go-analysis/          # 示例 Skill
    ├── SKILL.md
    └── references/
        └── GO_CONVENTIONS.md
```
