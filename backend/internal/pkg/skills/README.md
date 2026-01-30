# Skills 动态加载框架

基于 [Agent Skills Specification](https://agentskills.io/specification) 的技能框架，支持通过 Markdown 文件定义专业技能，动态加载并注入 LLM 上下文。

## 核心概念

### Skill 结构

每个 Skill 是一个目录，包含 `SKILL.md` 文件：

```
skill-name/
├── SKILL.md          # 必需：元数据 + 指令
├── scripts/          # 可选：可执行脚本
├── references/       # 可选：参考文档
└── assets/           # 可选：静态资源
```

### SKILL.md 格式

```markdown
---
name: skill-name
description: A description of what this skill does and when to use it.
license: MIT
compatibility: Requires Python 3.9+
metadata:
  author: example
  version: "1.0"
---

# 技能指令

详细的步骤说明、示例、最佳实践...
```

### Progressive Disclosure（渐进式披露）

- **L1 - 元数据**：启动时加载所有 Skills 的 name 和 description
- **L2 - 指令**：匹配 Skill 时加载 SKILL.md body
- **L3 - 资源**：按需加载 references/ 中的文件

## 快速开始

### 1. 初始化 Manager

```go
import "github.com/opendeepwiki/backend/internal/pkg/skills"

// 使用默认配置
manager, err := skills.NewManager(nil)
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()

// 或自定义配置
manager, err := skills.NewManager(&skills.Config{
    Dir:            "./my-skills",
    AutoReload:     true,
    ReloadInterval: 5 * time.Second,
})
```

### 2. 匹配 Skills 并注入 Prompt

```go
// 定义任务
task := skills.Task{
    Type:        "architecture",
    Description: "分析这个 Go 项目的架构",
    RepoType:    "go",
    Tags:        []string{"microservice"},
}

// 匹配并注入
newPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
if err != nil {
    log.Fatal(err)
}

// 查看匹配的 Skills
for _, m := range matches {
    fmt.Printf("Skill: %s, Score: %.0f%%, Reason: %s\n", 
        m.Skill.Name, m.Score*100, m.Reason)
}
```

### 3. 手动使用组件

```go
// 匹配 Skills
matches, err := manager.Matcher.Match(task)

// 构建 Skill 上下文
context, err := manager.Injector.BuildSkillContext(matches)

// 获取单个 Skill 内容
skill, body, err := manager.GetSkillContent("go-analysis")
```

## 组件说明

| 组件 | 职责 | 主要方法 |
|-----|------|---------|
| **Parser** | 解析 SKILL.md | `Parse()`, `ParseMetadata()`, `Validate()` |
| **Loader** | 加载 Skills | `LoadFromDir()`, `LoadFromPath()`, `GetBody()` |
| **Registry** | 管理 Skills | `Register()`, `Get()`, `List()`, `Enable()` |
| **Matcher** | 匹配 Skills | `Match()`, `MatchByDescription()` |
| **Injector** | 注入 Prompt | `InjectToPrompt()`, `BuildSkillContext()` |
| **Manager** | 整合管理 | `MatchAndInject()`, `ReloadAll()` |

## Skill 编写规范

### name 规范

- 只能包含小写字母、数字、连字符
- 不能以连字符开头或结尾
- 不能包含连续连字符
- 长度 1-64 字符
- 应与目录名一致

### description 规范

- 描述技能做什么和何时使用
- 包含关键词帮助匹配
- 长度 1-1024 字符
- 良好的示例：
  ```
  Extracts text and tables from PDF files, fills PDF forms. 
  Use when working with PDF documents or when user mentions PDFs.
  ```

### 指令编写建议

- 提供清晰的步骤说明
- 包含输入输出示例
- 说明常见边界情况
- 保持 main SKILL.md 在 500 行以内
- 详细参考文档放在 references/ 目录

## 配置

### 环境变量

- `SKILLS_DIR`: 指定 Skills 目录

### 配置选项

```go
type Config struct {
    Dir            string        // Skills 目录，默认 "./skills"
    AutoReload     bool          // 自动热加载，默认 true
    ReloadInterval time.Duration // 检查间隔，默认 5s
}
```

## 示例 Skills

### go-analysis

分析 Go 项目架构，识别模块依赖和代码组织。

适用场景：
- Go 仓库分析
- 架构文档生成
- 代码审查

### doc-generation

生成技术文档的最佳实践指南。

适用场景：
- 项目文档编写
- API 文档生成
- 开发者指南

## 测试

```bash
cd backend
go test ./internal/pkg/skills/... -v
```

## 参考

- [Agent Skills Specification](https://agentskills.io/specification)
- [Claude Custom Skills](https://support.claude.com/en/articles/12512198-how-to-create-custom-skills)
