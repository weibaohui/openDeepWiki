---
name: dependency-management
description: 解决章节间依赖。解析章节依赖关系，检测冲突，建议解决方案。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Dependency Management Skill

管理章节间的依赖关系，确保文档生成顺序正确。

## 使用场景

- OrchestratorAgent 调度章节生成
- 解决章节间的引用依赖
- 检测和解决冲突

## 功能能力

### 1. resolve_dependencies

解析章节依赖关系。

**输入：**
```yaml
chapter_plans:
  - id: "ch1"
    title: "项目概览"
    provides:
      - "项目背景"
      - "核心概念"
    
  - id: "ch2"
    title: "架构设计"
    depends_on:
      - concept: "核心概念"
        provided_by: "ch1"
    provides:
      - "整体架构"
      
  - id: "ch3"
    title: "核心模块"
    depends_on:
      - concept: "整体架构"
        provided_by: "ch2"
      - concept: "核心概念"
        provided_by: "ch1"
```

**输出：**
```yaml
dependency_graph:
  nodes:
    - id: "ch1"
      title: "项目概览"
      
    - id: "ch2"
      title: "架构设计"
      
    - id: "ch3"
      title: "核心模块"
      
  edges:
    - from: "ch1"
      to: "ch2"
      type: "provides"
      concept: "核心概念"
      
    - from: "ch2"
      to: "ch3"
      type: "provides"
      concept: "整体架构"
      
    - from: "ch1"
      to: "ch3"
      type: "provides"
      concept: "核心概念"

execution_order:
  - ["ch1"]  # 第一层：无依赖
  - ["ch2"]  # 第二层：依赖 ch1
  - ["ch3"]  # 第三层：依赖 ch1, ch2
```

### 2. detect_conflicts

检测章节间冲突。

**冲突类型：**
- 循环依赖
- 重复定义
- 矛盾描述
- 资源竞争

**输出：**
```yaml
conflicts:
  - type: "circular_dependency"
    description: "ch2 依赖 ch3，ch3 又依赖 ch2"
    cycle: ["ch2", "ch3", "ch2"]
    severity: "high"
    suggestion: "提取共同依赖到 ch1，或合并 ch2 和 ch3"
    
  - type: "duplicate_definition"
    concept: "路由器"
    definitions:
      - chapter: "ch2"
        section: "架构组件"
        definition: "路由器是 HTTP 请求入口"
      - chapter: "ch3"
        section: "路由实现"
        definition: "路由器负责管理路由注册"
    severity: "medium"
    suggestion: "在 ch1 统一定义，后续章节引用"
    
  - type: "contradictory_description"
    concept: "路由匹配算法复杂度"
    descriptions:
      - chapter: "ch2"
        content: "时间复杂度 O(n)"
      - chapter: "ch4"
        content: "时间复杂度 O(log n)"
    severity: "high"
    suggestion: "统一描述，以代码分析为准"
```

### 3. suggest_resolution

建议解决方案。

**策略：**
- 重新排序
- 提取公共内容
- 合并章节
- 使用占位符

**输出：**
```yaml
resolution_plan:
  strategy: "reorder_and_extract"
  
  steps:
    - action: "extract"
      content: "核心概念定义"
      from: ["ch2", "ch3"]
      to: "ch1"
      
    - action: "reorder"
      new_order: ["ch1", "ch2", "ch3", "ch4"]
      
    - action: "placeholder"
      chapter: "ch3"
      references:
        - concept: "中间件机制"
          placeholder: "[中间件机制将在第4章详细说明]"
  
  expected_outcome: "消除循环依赖，统一概念定义"
```

## 完整输出格式

```yaml
DependencyResolution:
  dependency_graph: object
  execution_order: array
  
  conflicts:
    circular: array
    duplicates: array
    contradictions: array
    
  resolution_plan:
    strategy: string
    steps: array
    expected_outcome: string
```

## 使用示例

```yaml
# 在 OrchestratorAgent 中使用
skills:
  - dependency-management

task:
  name: 解析章节依赖
  steps:
    - action: dependency-management.resolve
      input:
        chapter_plans: "{{all_chapter_plans}}"
      output: dependency_resolution
```

## 依赖

无

## 最佳实践

1. 提前定义章节间的依赖关系
2. 循环依赖需要重构
3. 核心概念应在早期章节定义
4. 使用占位符处理跨章节引用
