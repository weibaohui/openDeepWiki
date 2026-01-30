---
name: context-management
description: 维护全局上下文和记忆。在Agent间共享上下文，管理概念定义表。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Context Management Skill

管理文档生成的全局上下文和共享记忆。

## 使用场景

- OrchestratorAgent 维护全局上下文
- 在 Agent 间共享信息
- 确保术语一致性

## 功能能力

### 1. update_global_context

更新全局上下文。

**上下文内容：**
```yaml
global_context:
  project:
    name: "示例项目"
    type: "web_service"
    language: "Go"
    
  chapters:
    - id: "ch1"
      title: "项目概览"
      summary: "介绍项目背景、核心功能..."
      key_concepts:
        - "路由器"
        - "中间件"
        
    - id: "ch2"
      title: "架构设计"
      summary: "描述分层架构..."
      
  concept_definitions:
    路由器:
      definition: "负责管理路由注册和请求分发的核心组件"
      first_defined_in: "ch1"
      aliases: ["Router"]
      
    中间件:
      definition: "在请求处理前后执行的可插拔组件"
      first_defined_in: "ch1"
      aliases: ["Middleware"]
      
  terminology_table:
    路由器: "Router"
    路由: "Route"
    处理器: "Handler"
    中间件: "Middleware"
```

**更新操作：**
```yaml
context_update:
  operation: "add_chapter"  # add_chapter, update_concept, add_alias
  data:
    chapter:
      id: "ch3"
      title: "核心模块"
      summary: "..."
```

### 2. manage_concepts

管理概念定义表。

**操作：**
- 添加新概念
- 更新定义
- 添加别名
- 查询概念

**输出：**
```yaml
concept_management:
  added:
    - term: "前缀树"
      definition: "一种树形数据结构，用于高效字符串匹配"
      
  updated:
    - term: "路由器"
      old_definition: "..."
      new_definition: "..."
      reason: "补充了负载均衡功能"
      
  aliases_added:
    - term: "路由器"
      alias: "路由管理器"
      
  conflicts_detected:
    - term: "路由"
      conflict: "有时指 Route，有时指 Router"
      suggestion: "统一使用'路由'表示 Route，'路由器'表示 Router"
```

### 3. share_context

在 Agent 间共享上下文。

**输入：**
```yaml
source_agent: "ArchitectAgent"
target_agents: ["ExplorerAgent", "WriterAgent"]
context_to_share:
  - repo_meta
  - doc_outline
  - concept_definitions
```

**输出：**
```yaml
share_result:
  success: true
  shared_with:
    - agent: "ExplorerAgent"
      context_id: "ctx_001_exp"
      
    - agent: "WriterAgent"
      context_id: "ctx_001_wri"
      
  shared_data:
    repo_meta: object
    doc_outline: object
    concept_count: 15
```

## 完整输出格式

```yaml
GlobalContext:
  project: object
  chapters: array
  concept_definitions: object
  terminology_table: object
  
  version: int
  last_updated: timestamp
  updated_by: string
```

## 使用示例

```yaml
# 在 OrchestratorAgent 中使用
skills:
  - context-management

task:
  name: 管理全局上下文
  steps:
    - action: context-management.update
      input:
        operation: "add_chapter"
        data: "{{new_chapter}}"
      output: updated_context
      
    - action: context-management.share
      input:
        target_agents: ["ExplorerAgent"]
        context: "{{global_context}}"
      output: share_result
```

## 依赖

无

## 最佳实践

1. 核心概念在首次出现时定义
2. 维护术语对照表
3. 定期同步上下文到所有相关 Agent
4. 版本控制上下文变更
