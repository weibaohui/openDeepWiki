---
name: style-check
description: 检查写作风格和格式规范。检查 Markdown 格式、代码块样式、标题层级。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: quality-assurance
  priority: P2
---

# Style Check Skill

检查文档的格式规范和写作风格。

## 使用场景

- ReviewerAgent 检查格式规范
- EditorAgent 统一文档风格
- 自动化格式检查

## 功能能力

### 1. check_markdown_format

检查 Markdown 格式。

**检查项：**
- 标题层级连续（# → ## → ###）
- 列表标记一致（- 或 *）
- 代码块指定语言
- 链接格式正确
- 图片有 alt 文本

**输出：**
```yaml
markdown_report:
  issues:
    - type: "heading_skip"
      location: "第10行"
      description: "从 # 直接跳到 ###，缺少 ##"
      suggestion: "添加二级标题或使用 ## 而非 ###"
      
    - type: "code_block_no_language"
      location: "第45-50行"
      description: "代码块未指定语言"
      suggestion: "添加 go 语言标识：```go"
      
    - type: "inconsistent_list_marker"
      location: "第20-30行"
      description: "混用 - 和 * 作为列表标记"
      suggestion: "统一使用 -"
      
    - type: "broken_link"
      location: "第55行"
      description: "链接格式错误"
      content: "[链接](http://example.com"  # 缺少右括号
      suggestion: "修正为 [链接](http://example.com)"
```

### 2. check_code_block_style

检查代码块样式。

**检查项：**
- 语言标识正确
- 代码缩进一致
- 无尾随空格
- 行长度适中（< 100 字符）

**输出：**
```yaml
code_style_report:
  issues:
    - type: "incorrect_indentation"
      location: "代码块1，第3行"
      description: "使用了 2 空格缩进，应为 4 空格或 Tab"
      
    - type: "trailing_whitespace"
      location: "代码块2，第5行"
      description: "行尾有多余空格"
      
    - type: "line_too_long"
      location: "代码块1，第8行"
      description: "行长度 120 字符，建议不超过 100"
      
  suggestions:
    - "函数定义后添加空行"
    - "复杂逻辑添加注释"
```

### 3. check_heading_hierarchy

检查标题层级。

**输出：**
```yaml
heading_report:
  structure:
    - level: 1
      title: "路由系统"
      line: 1
      
    - level: 2
      title: "整体架构"
      line: 10
      
    - level: 2
      title: "核心实现"
      line: 25
      
    - level: 3
      title: "路由匹配"
      line: 30
      
    - level: 3
      title: "参数提取"
      line: 45
      
  issues:
    - type: "missing_h2"
      location: "第30行前"
      description: "从 # 路由系统 直接跳到 ### 路由匹配"
      suggestion: "添加 ## 章节标题"
      
    - type: "orphan_heading"
      location: "第60行"
      description: "四级标题 #### 没有上级三级标题"
      suggestion: "提升为三级标题或添加上级标题"
```

## 写作风格指南

### 1. 标题规范

```markdown
<!-- 好的写法 -->
## 路由匹配算法

### 前缀树结构

#### 节点定义

<!-- 不好的写法 -->
## 路由匹配算法

#### 前缀树结构  <!-- 跳过三级 -->
```

### 2. 代码块规范

```markdown
<!-- 好的写法 -->
```go
func main() {
    fmt.Println("Hello")
}
```

<!-- 不好的写法 -->
```
func main() {
    fmt.Println("Hello")
}
```
<!-- 缺少语言标识 -->
```

### 3. 列表规范

```markdown
<!-- 好的写法 -->
- 第一项
- 第二项
  - 子项 1
  - 子项 2
- 第三项

<!-- 不好的写法 -->
- 第一项
* 第二项  <!-- 混用标记 -->
  - 子项 1
  * 子项 2  <!-- 混用标记 -->
```

## 完整输出格式

```yaml
StyleReport:
  format_score: float
  issues: array
  suggestions: array
  
  auto_fixable: boolean
  fixed_version: string  # 如果可以自动修复
```

## 使用示例

```yaml
# 在 ReviewerAgent 中使用
skills:
  - style-check

task:
  name: 检查格式
  steps:
    - action: style-check.verify
      input:
        document_content: "{{section_draft}}"
        style_guide: "{{global_context.style_guide}}"
      output: style_report
```

## 依赖

无

## 最佳实践

1. 保持标题层级连续
2. 代码块始终指定语言
3. 使用统一的列表标记
4. 行长度不超过 100 字符
5. 定期运行格式检查
