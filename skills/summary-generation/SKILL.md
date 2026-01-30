---
name: summary-generation
description: 摘要生成。生成章节摘要、TL;DR。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Summary Generation Skill

为文档生成摘要和 TL;DR。

## 使用场景

- EditorAgent 为章节生成摘要
- 生成文档概览
- 创建快速参考

## 功能能力

### 1. generate_chapter_summary

生成章节摘要。

**输入：**
```yaml
chapter_content: |
  # 路由匹配机制
  
  ## 概念速览
  动态路由匹配是 Web 框架的核心功能...
  
  ## 代码位置
  核心实现在 router.go...
  
  ## 执行流程
  匹配流程分为以下步骤...
  
target_length: 200  # 目标字数
```

**输出：**
```yaml
chapter_summary:
  short: |
    本章介绍了路由匹配机制的实现。使用前缀树数据结构实现 O(n) 
    时间复杂度的路由匹配，支持静态路由和动态参数。
  
  detailed: |
    本章详细介绍了路由匹配机制的实现原理和代码细节。
    
    **核心内容：**
    1. **概念速览**：动态路由匹配是 Web 框架的核心功能，
       负责将 URL 映射到对应的处理函数。
    
    2. **实现位置**：核心代码位于 `router.go`，
       主要函数为 `matchRoute`。
    
    3. **执行流程**：路径分割 → 前缀树遍历 → 参数提取 → 返回匹配结果。
    
    4. **设计考量**：使用前缀树而非正则，保证匹配效率和可读性。
  
  key_points:
    - "前缀树数据结构"
    - "O(n) 时间复杂度"
    - "支持动态参数"
    - "router.go 核心实现"
```

### 2. generate_tldr

生成 TL;DR（太长不看版）。

**输出：**
```yaml
tldr:
  version: |
    **TL;DR**: 路由匹配使用前缀树实现 O(n) 复杂度的查找，
    代码在 `router.go`。
  
  extended: |
    **TL;DR**
    
    - **是什么**：将 URL 映射到处理函数的机制
    - **在哪**：`router.go` 的 `matchRoute` 函数
    - **怎么做**：前缀树遍历，静态优先于动态匹配
    - **为什么**：O(n) 时间复杂度，比正则匹配高效
    
    **一句话总结**：用前缀树高效匹配 URL 到处理器。
```

## 摘要类型

| 类型 | 长度 | 用途 |
|-----|------|-----|
| One-liner | 1 句话 | 章节标题 tooltip |
| Short | 2-3 句话 | 目录预览 |
| Detailed | 1 段落 | 章节开头摘要 |
| TL;DR | 列表 | 快速参考 |

## 使用示例

```yaml
# 在 EditorAgent 中使用
skills:
  - summary-generation

task:
  name: 生成摘要
  steps:
    - action: summary-generation.generate
      input:
        chapter_content: "{{chapter_content}}"
        types: ["short", "detailed", "tldr"]
      output: summaries
```

## 依赖

- generation.llm_generate

## 最佳实践

1. 摘要要涵盖核心要点
2. 使用简洁的语言
3. TL;DR 放在章节开头
4. 详细摘要帮助快速回顾
