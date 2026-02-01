---
name: content-assembly
description: 内容组装。组装各小节为完整章节，生成目录，添加导航链接。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Content Assembly Skill

将多个小节组装成完整的章节文档。

## 使用场景

- EditorAgent 组装章节
- 生成目录和导航
- 统一文档格式

## 功能能力

### 1. assemble_sections

组装各小节。

**输入：**
```yaml
sections:
  - id: "sec1"
    title: "概念速览"
    content: "..."
    order: 1
    
  - id: "sec2"
    title: "代码位置"
    content: "..."
    order: 2
    
  - id: "sec3"
    title: "执行流程"
    content: "..."
    order: 3
    
chapter_metadata:
  title: "路由匹配机制"
  id: "ch2-sec1"
  order: 1
```

**输出：**
```yaml
chapter_document:
  title: "路由匹配机制"
  content: |
    # 路由匹配机制
    
    ## 目录
    
    - [概念速览](#概念速览)
    - [代码位置](#代码位置)
    - [执行流程](#执行流程)
    
    ---
    
    ## 概念速览
    
    [内容]
    
    ## 代码位置
    
    [内容]
    
    ## 执行流程
    
    [内容]
    
    ---
    
    ## 相关章节
    
    - [上一章：架构设计](../architecture.md)
    - [下一章：中间件机制](../middleware.md)
  
  toc:
    - level: 2
      title: "概念速览"
      anchor: "概念速览"
    - level: 2
      title: "代码位置"
      anchor: "代码位置"
    - level: 2
      title: "执行流程"
      anchor: "执行流程"
```

### 2. generate_toc

生成目录。

**输出：**
```yaml
toc:
  markdown: |
    ## 目录
    
    - [概念速览](#概念速览)
    - [代码位置](#代码位置)
      - [核心文件](#核心文件)
      - [辅助文件](#辅助文件)
    - [执行流程](#执行流程)
    
  structure:
    - level: 2
      title: "概念速览"
      anchor: "概念速览"
      
    - level: 2
      title: "代码位置"
      anchor: "代码位置"
      children:
        - level: 3
          title: "核心文件"
          anchor: "核心文件"
        - level: 3
          title: "辅助文件"
          anchor: "辅助文件"
```

### 3. add_navigation

添加导航链接。

**导航类型：**
- 上一章/下一章
- 相关章节
- 返回目录
- 页内锚点

**输出：**
```yaml
navigation:
  header: |
    [← 返回目录](../README.md)
    
  footer: |
    ---
    
    **导航**
    
    [← 上一章：架构设计](./architecture.md) | [下一章：中间件机制](./middleware.md) →]
    
  related:
    - title: "相关章节"
      links:
        - ["HTTP 处理流程", "./http-handler.md"]
        - ["性能优化", "./performance.md"]
```

## 完整输出格式

```yaml
ChapterDocument:
  title: string
  content: string
  toc: object
  navigation: object
  
  metadata:
    section_count: int
    word_count: int
    code_block_count: int
    diagram_count: int
```

## 使用示例

```yaml
# 在 EditorAgent 中使用
skills:
  - content-assembly

task:
  name: 组装章节
  steps:
    - action: content-assembly.assemble
      input:
        sections: "{{all_sections}}"
        chapter_metadata: "{{chapter_info}}"
      output: chapter_document
```

## 依赖

- style-check（可选，用于格式检查）

## 最佳实践

1. 确保小节顺序正确
2. 生成清晰的目录
3. 添加必要的导航链接
4. 统一文档头部和尾部格式
