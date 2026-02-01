---
name: transition-optimization
description: 过渡优化。添加过渡段落，平滑段落衔接。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Transition Optimization Skill

优化文档中小节之间的过渡，使内容衔接更流畅。

## 使用场景

- EditorAgent 优化章节连贯性
- 改善阅读体验
- 确保逻辑流畅

## 功能能力

### 1. add_transition_paragraphs

添加过渡段落。

**输入：**
```yaml
sections:
  - title: "概念速览"
    content: |
      动态路由匹配是 Web 框架的核心功能...
      [内容结束]
      
  - title: "代码位置"
    content: |
      路由匹配的核心实现在 router.go 文件中...
      [内容开始]
```

**输出：**
```yaml
sections_with_transitions:
  - title: "概念速览"
    content: |
      动态路由匹配是 Web 框架的核心功能...
      [内容结束]
    
  - transition: |
      了解了动态路由匹配的基本概念后，
      我们来看看它在代码库中的具体实现位置。
      
  - title: "代码位置"
    content: |
      路由匹配的核心实现在 router.go 文件中...
      [内容开始]
```

### 2. smooth_flow

平滑段落衔接。

**分析维度：**
- 主题连贯性
- 逻辑递进
- 指代清晰
- 语气一致

**输出：**
```yaml
flow_analysis:
  transitions:
    - from: "概念速览"
      to: "代码位置"
      quality: "good"
      connection: "概念→实现"
      
    - from: "代码位置"
      to: "执行流程"
      quality: "needs_improvement"
      issue: "缺少从静态代码到动态流程的过渡"
      suggestion: "添加：'知道了代码在哪里，接下来我们看看这些代码是如何执行的'"
      
    - from: "执行流程"
      to: "设计考量"
      quality: "good"
      connection: "实现→设计意图"
      
  improvements:
    - location: "代码位置→执行流程"
      original: "[直接开始]"
      improved: |
        知道了核心代码的位置，接下来我们深入看看
        `matchRoute` 函数是如何一步步完成路由匹配的。
```

## 过渡句模板

### 概念→实现

- "了解了 {概念} 的基本原理，我们来看看具体的代码实现。"
- "{概念} 的核心思想比较抽象，下面通过代码来具体理解。"

### 实现→流程

- "知道了代码在哪里，接下来看看这些代码是如何执行的。"
- "了解了整体结构，下面详细分析执行流程。"

### 流程→设计

- "了解了实现细节，我们再来思考为什么要这样设计。"
- "通过上面的分析可以看出，这种设计有以下几个考虑..."

### 具体→抽象

- "从上面的具体实现中，我们可以总结出..."
- "这个例子展示了 {概念} 的典型用法。"

## 使用示例

```yaml
# 在 EditorAgent 中使用
skills:
  - transition-optimization

task:
  name: 优化过渡
  steps:
    - action: transition-optimization.smooth
      input:
        sections: "{{sections}}"
      output: optimized_sections
```

## 依赖

无

## 最佳实践

1. 每个小节之间都应有适当的过渡
2. 过渡句要简洁自然
3. 保持叙述视角一致
4. 避免过于生硬的教学语气
