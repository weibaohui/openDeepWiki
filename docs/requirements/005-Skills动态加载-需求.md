# 005-Skills动态加载-需求.md

## 1. 背景（Why）

当前系统的 LLM 调用缺乏领域特定的知识和工作流程指导。为了让 LLM 更好地完成特定任务（如代码分析、文档生成、特定领域处理），需要引入 **Skills（技能）机制**。

Skills 是 Agent Skills 规范的实现，允许通过 Markdown 文件定义专业技能，包含：
- **元数据**：技能名称、描述、使用场景
- **指令**：详细的步骤说明、示例、最佳实践
- **资源**：参考文档、可执行脚本、模板文件

当 LLM 处理特定任务时，系统根据任务内容匹配合适的 Skills，将 Skill 的指令注入到 LLM 上下文中，指导 LLM 更好地完成任务。

### 目标

- [ ] 实现 Agent Skills 规范兼容的 Skills 框架
- [ ] 支持从 `SKILL.md` 文件动态加载 Skills
- [ ] Skills 可在运行时热加载/更新，无需重启服务
- [ ] 支持智能匹配：根据任务内容自动选择合适的 Skills
- [ ] Skills 内容可注入 LLM 上下文，指导任务执行

### 非目标

- [ ] 不实现脚本执行（scripts/ 目录内容由外部处理）
- [ ] 不实现 Skill 的版本管理
- [ ] 不涉及前端 UI 实现
- [ ] 不实现 Skill 之间的显式依赖

---

## 2. 核心概念定义

### 2.1 Skill（技能）

Skill 是一个包含专业知识和工作流程的单元，以目录形式组织：

```
skill-name/
├── SKILL.md          # 必需：元数据 + 指令
├── scripts/          # 可选：可执行脚本（Python/Bash/JS等）
├── references/       # 可选：参考文档
└── assets/           # 可选：静态资源（模板、数据文件）
```

**SKILL.md 结构**：
- **YAML Frontmatter**（元数据）：name, description, license, compatibility, metadata
- **Markdown Body**（指令）：步骤说明、示例、最佳实践

### 2.2 Skill Registry（技能注册中心）

管理所有已加载的 Skills：
- 维护 Skills 元数据索引（用于匹配）
- 管理 Skills 的启用/禁用状态
- 提供 Skills 查询和检索能力

### 2.3 Skill Matcher（技能匹配器）

根据任务内容选择合适的 Skills：
- 分析任务描述、类型、上下文
- 匹配 Skills 的 description 关键字
- 返回匹配的 Skills 列表

### 2.4 Progressive Disclosure（渐进式披露）

Skills 内容分层加载策略：

| 层级 | 内容 | 加载时机 | 大小建议 |
|-----|------|---------|---------|
| L1 | 元数据（name, description） | 系统启动时 | ~100 tokens |
| L2 | 指令（SKILL.md body） | Skill 被选中时 | <5000 tokens |
| L3 | 资源（references/等） | 按需加载 | 根据需求 |

---

## 3. 功能需求清单（Checklist）

### 3.1 Skill 解析

- [ ] 解析 `SKILL.md` 文件的 YAML frontmatter
- [ ] 提取 Markdown body 作为指令内容
- [ ] 校验 name 格式（小写字母、数字、连字符，最多64字符）
- [ ] 校验 description 长度（1-1024字符）
- [ ] 支持可选字段：license, compatibility, metadata, allowed-tools

### 3.2 Skill 目录结构

- [ ] 加载单个 Skill 目录
- [ ] 支持 `scripts/` 子目录识别
- [ ] 支持 `references/` 子目录识别
- [ ] 支持 `assets/` 子目录识别
- [ ] 支持相对路径引用其他文件

### 3.3 动态加载机制

- [ ] 默认目录：`./skills`（与可执行文件同级）
- [ ] 支持环境变量 `SKILLS_DIR` 指定目录
- [ ] 支持配置文件指定 `skills.dir`
- [ ] 系统启动时加载所有 Skills 元数据
- [ ] 监听目录变化，热加载/更新/卸载 Skills
- [ ] 支持手动刷新 API

### 3.4 Skill Registry

- [ ] `Register(skill)`: 注册 Skill
- [ ] `Unregister(name)`: 注销 Skill
- [ ] `Enable(name)`: 启用 Skill
- [ ] `Disable(name)`: 禁用 Skill
- [ ] `Get(name)`: 获取指定 Skill
- [ ] `List()`: 列出所有 Skills（仅元数据）
- [ ] `ListEnabled()`: 列出启用的 Skills
- [ ] `GetInstructions(name)`: 获取 Skill 完整指令

### 3.5 Skill Matcher

- [ ] 基于任务描述匹配 Skills
- [ ] 支持关键词匹配
- [ ] 支持任务类型匹配
- [ ] 返回匹配度和排序
- [ ] 支持强制指定 Skills

### 3.6 LLM 集成

- [ ] 将匹配的 Skills 注入 System Prompt
- [ ] 支持多个 Skills 组合使用
- [ ] 支持 Skill 指令与原有 Prompt 融合
- [ ] 记录使用的 Skills（用于调试）

---

## 4. 数据结构

### 4.1 Skill 结构

```go
// Skill 技能定义
type Skill struct {
    // 元数据（始终加载）
    Name           string            `yaml:"name" json:"name"`
    Description    string            `yaml:"description" json:"description"`
    License        string            `yaml:"license,omitempty" json:"license,omitempty"`
    Compatibility  string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
    Metadata       map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
    AllowedTools   string            `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
    
    // 指令（按需加载）
    Instructions   string            `json:"instructions,omitempty"` // SKILL.md body
    
    // 路径信息
    Path           string            `json:"path"`           // Skill 目录绝对路径
    SkillMDPath    string            `json:"skill_md_path"`  // SKILL.md 文件路径
    
    // 资源
    HasScripts     bool              `json:"has_scripts"`
    HasReferences  bool              `json:"has_references"`
    HasAssets      bool              `json:"has_assets"`
    
    // 状态
    Enabled        bool              `json:"enabled"`
    LoadedAt       time.Time         `json:"loaded_at"`
}
```

### 4.2 Skill 加载结果

```go
// SkillLoadResult 加载结果
type SkillLoadResult struct {
    Skill   *Skill
    Error   error
    Action  string // "created", "updated", "unchanged"
}
```

---

## 5. 使用场景

### 场景 1：代码仓库分析

```
任务：分析 Go 项目的架构
    ↓
Matcher 匹配到 "go-analysis" Skill
    ↓
加载 SKILL.md body（Go 项目分析步骤、最佳实践）
    ↓
注入 LLM 上下文：
  - System Prompt 原有内容
  + "## Go 分析技能\n[SKILL.md body]\n"
    ↓
LLM 按照 Skill 指导生成更准确的架构文档
```

### 场景 2：文档生成

```
任务：生成 API 文档
    ↓
Matcher 匹配到 "api-doc" Skill
    ↓
加载 references/OPENAPI.md 作为参考
    ↓
LLM 根据 Skill 指令和参考文档生成规范 API 文档
```

### 场景 3：多技能组合

```
任务：分析 Python 微服务并生成部署文档
    ↓
Matcher 匹配到：
  - "python-analysis" Skill
  - "microservice-patterns" Skill
  - "deployment-guide" Skill
    ↓
组合多个 Skills 的指令
    ↓
LLM 综合多个技能的指导完成任务
```

---

## 6. 接口设计

### 6.1 Skill Parser

```go
// Parser Skill 解析器
type Parser interface {
    // Parse 解析 Skill 目录
    Parse(skillPath string) (*Skill, error)
    
    // ParseMetadata 仅解析元数据（快速）
    ParseMetadata(skillPath string) (*Skill, error)
    
    // Validate 校验 Skill 有效性
    Validate(skill *Skill) error
}
```

### 6.2 Skill Loader

```go
// Loader Skill 加载器
type Loader interface {
    // LoadFromDir 从目录加载所有 Skills
    LoadFromDir(dir string) ([]*SkillLoadResult, error)
    
    // LoadFromPath 加载单个 Skill
    LoadFromPath(path string) (*Skill, error)
    
    // Reload 重新加载 Skill
    Reload(name string) (*Skill, error)
    
    // Unload 卸载 Skill
    Unload(name string) error
}
```

### 6.3 Skill Matcher

```go
// Matcher Skill 匹配器
type Matcher interface {
    // Match 根据任务匹配 Skills
    Match(task Task) ([]*SkillMatch, error)
    
    // MatchByDescription 根据描述匹配
    MatchByDescription(description string) ([]*SkillMatch, error)
    
    // GetAllEnabled 获取所有启用的 Skills
    GetAllEnabled() []*Skill
}

// SkillMatch 匹配结果
type SkillMatch struct {
    Skill      *Skill
    Score      float64  // 匹配分数 0-1
    Reason     string   // 匹配原因
}

// Task 任务定义
type Task struct {
    Type        string   // 任务类型
    Description string   // 任务描述
    RepoType    string   // 仓库类型（go/python等）
    Tags        []string // 标签
}
```

### 6.4 Skill Injector

```go
// Injector Skill 注入器
type Injector interface {
    // InjectToPrompt 将 Skills 注入到 Prompt
    InjectToPrompt(systemPrompt string, skills []*Skill) (string, error)
    
    // BuildSkillContext 构建 Skills 上下文
    BuildSkillContext(skills []*Skill) string
}
```

---

## 7. 错误处理

| 错误类型 | 说明 | 处理方式 |
|---------|------|---------|
| `ErrSkillNotFound` | Skill 不存在 | 返回错误 |
| `ErrInvalidMetadata` | 元数据无效 | 记录日志，跳过该 Skill |
| `ErrInvalidName` | name 格式错误 | 记录日志，跳过该 Skill |
| `ErrSkillLoadFailed` | 加载失败 | 记录日志，继续加载其他 |
| `ErrSkillDirNotFound` | Skills 目录不存在 | 创建空目录，继续启动 |

---

## 8. 配置

```yaml
# config.yaml
skills:
  dir: "./skills"              # Skills 目录
  auto_reload: true            # 自动热加载
  reload_interval: 5           # 检查间隔（秒）
  max_skill_tokens: 5000       # 单个 Skill 最大 token 数
  default_skills:              # 默认启用的 Skills
    - "code-analysis"
    - "doc-generation"
```

环境变量：
- `SKILLS_DIR`: 指定 Skills 目录
- `SKILLS_AUTO_RELOAD`: 是否自动热加载

---

## 9. 验收标准

### 9.1 功能验收

- [ ] 如果创建 `skills/my-skill/SKILL.md`，系统应自动加载
- [ ] 如果修改 SKILL.md，系统应在 5 秒内更新
- [ ] 如果删除 Skill 目录，系统应自动卸载
- [ ] `List()` 应返回所有已加载 Skills 的元数据
- [ ] `GetInstructions(name)` 应返回完整的 SKILL.md body
- [ ] Matcher 应根据任务描述返回匹配的 Skills
- [ ] Injector 应将 Skills 内容正确注入 Prompt

### 9.2 性能验收

- [ ] 加载 50 个 Skills 的元数据应在 100ms 内完成
- [ ] 单个 Skill 的完整加载应在 50ms 内完成
- [ ] 匹配操作应在 10ms 内完成

### 9.3 规范验收

- [ ] Skill name 必须符合规范（小写、数字、连字符）
- [ ] YAML frontmatter 必须正确解析
- [ ] 目录结构必须符合 Agent Skills 规范

---

## 10. 交付物

- [ ] Skills 核心接口定义（parser, loader, matcher, injector）
- [ ] Skill Registry 实现
- [ ] Skill Parser 实现（YAML frontmatter + Markdown）
- [ ] Skill Loader 实现（目录扫描、热加载）
- [ ] Skill Matcher 实现（关键词匹配）
- [ ] Skill Injector 实现（Prompt 注入）
- [ ] 示例 Skills（code-analysis, doc-generation）
- [ ] 单元测试
- [ ] 使用文档

---

## 11. 参考规范

- [Agent Skills Specification](https://agentskills.io/specification)
- [Claude Custom Skills](https://support.claude.com/en/articles/12512198-how-to-create-custom-skills)
