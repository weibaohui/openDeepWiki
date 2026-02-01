---
name: narrative-flow
description: 组织技术叙事结构。组织"What-Where-How-Why"结构，添加过渡段落。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: writing
  priority: P1
---

# Narrative Flow Skill

组织技术文档的叙事结构，确保内容连贯、逻辑清晰。

## 使用场景

- WriterAgent 组织小节内容
- 优化文档的可读性
- 确保技术内容有逻辑地展开

## 功能能力

### 1. organize_structure

组织 "What-Where-How-Why" 结构。

**标准技术写作结构：**

```yaml
structure:
  what:
    title: "概念速览"
    content: "这是什么？解决什么问题？"
    length: "2-3 段"
    
  where:
    title: "代码位置"
    content: "相关代码在哪里？文件、函数位置"
    length: "1-2 段 + 代码引用"
    
  how:
    title: "执行流程"
    content: "如何工作？详细步骤"
    length: "主体部分，可细分多个步骤"
    
  why:
    title: "设计意图"
    content: "为什么这样设计？权衡考虑"
    length: "2-3 段"
```

**输出：**
```yaml
organized_content:
  sections:
    - type: "what"
      title: "概念速览"
      content: |
        动态路由匹配是 Web 框架的核心功能，负责将 HTTP 请求 URL 
        映射到对应的处理函数。与静态路由不同，动态路由支持路径参数，
        如 `/user/:id` 可以匹配 `/user/123`。
        
    - type: "where"
      title: "代码位置"
      content: |
        路由匹配的核心实现在 `router.go` 文件的 `matchRoute` 函数：
        
        ```go
        // router.go:45
        func (r *Router) matchRoute(path string) (*Route, Params)
        ```
        
    - type: "how"
      title: "执行流程"
      content: |
        匹配流程分为以下几个步骤：
        
        **1. 路径分割**
        
        首先将请求路径按 `/` 分割...
        
        **2. 前缀树遍历**
        
        从根节点开始，逐级匹配路径段...
        
    - type: "why"
      title: "设计意图"
      content: |
        选择前缀树而非正则匹配，主要考虑以下因素：
        
        - **性能**：前缀树匹配时间复杂度为 O(n)，正则匹配最坏可达 O(2^n)
        - **可读性**：树形结构更直观，便于调试
        - **扩展性**：易于添加新的匹配规则
```

### 2. add_transitions

添加过渡段落。

**输入：**
```yaml
sections:
  - title: "路径分割"
    content: "..."
  - title: "前缀树遍历"
    content: "..."
  - title: "参数提取"
    content: "..."
```

**输出：**
```yaml
sections_with_transitions:
  - title: "路径分割"
    content: "..."
    
  - transition: |
      分割完成后，我们就得到了一个路径段数组。
      接下来，路由器会使用这些数据在前缀树中查找匹配。
      
  - title: "前缀树遍历"
    content: "..."
    
  - transition: |
      当找到匹配的叶子节点时，我们还需要处理路径中的动态参数。
      
  - title: "参数提取"
    content: "..."
```

### 3. maintain_coherence

保持叙述连贯性。

**检查项：**
- 代词指代清晰（这个/那个/它）
- 专业术语一致
- 时态一致（优先使用现在时）
- 逻辑连接词使用恰当

**输出：**
```yaml
coherence_report:
  issues:
    - type: "ambiguous_reference"
      location: "第3段第2句"
      suggestion: "将'它'改为'前缀树'"
      
    - type: "inconsistent_tense"
      location: "第5段"
      suggestion: "统一使用现在时'使用'而非过去时'使用了'"
      
  improved_version: "..."
```

### 4. apply_inverted_pyramid

应用倒金字塔结构（先结论后细节）。

**转换示例：**

```markdown
<!-- 原内容（金字塔结构） -->
路由匹配首先将路径分割，然后构建前缀树，
前缀树的每个节点代表路径段...
[大量细节]
...
因此，路由匹配的时间复杂度是 O(n)。

<!-- 改进后（倒金字塔结构） -->
路由匹配使用前缀树实现 O(n) 时间复杂度的查找。

具体来说，系统首先将路径分割为多个段，
然后在前缀树中逐级匹配...
[详细说明]
```

## 写作模板

### 算法解释模板

```markdown
## {算法名称}

### 概念速览
{一句话概括算法的目的和核心思想}

### 复杂度
- **时间复杂度**: {O(?))}
- **空间复杂度**: {O(?)}

### 执行流程
{步骤说明}

### 应用场景
{何时使用此算法}

### 与其他方案对比
{优缺点对比}
```

### 架构组件模板

```markdown
## {组件名称}

### 职责
{这个组件负责什么}

### 位置
{代码位置}

### 协作关系
- 与 {组件A}: {关系描述}
- 与 {组件B}: {关系描述}

### 关键实现
{核心代码解释}

### 设计考量
{为什么这样设计}
```

## 使用示例

```yaml
# 在 WriterAgent 中使用
skills:
  - narrative-flow

task:
  name: 组织内容结构
  steps:
    - action: narrative-flow.organize
      input:
        raw_content: "{{draft_content}}"
        structure_type: "what-where-how-why"
        target_audience: "中级开发者"
      output: organized_content
```

## 依赖

- generation.llm_generate

## 最佳实践

1. 根据读者水平调整详细程度
2. 技术概念首次出现时要定义
3. 复杂流程使用编号列表
4. 相关概念之间添加过渡句
