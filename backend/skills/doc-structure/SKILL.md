---
name: doc-structure
description: 根据项目类型生成文档大纲模板。为不同项目类型生成三级文档大纲。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: content-planning
  priority: P0
---

# Doc Structure Skill

根据仓库类型和结构分析结果生成文档大纲。

## 使用场景

- ArchitectAgent 生成文档骨架
- 为新仓库创建文档结构
- 根据分支类型调整文档侧重点

## 功能能力

### 1. select_template

根据仓库类型选择模板。

**模板类型：**
- `web_service` - Web 服务/API 项目
- `cli_tool` - 命令行工具
- `algorithm_lib` - 算法/工具库
- `frontend_app` - 前端应用
- `fullstack_app` - 全栈应用
- `microservice` - 微服务架构
- `sdk_library` - SDK/客户端库

### 2. generate_outline

生成三级文档大纲。

**输出格式：**
```yaml
doc_outline:
  chapters:
    - id: "ch1"
      title: "项目概览"
      order: 1
      sections:
        - id: "ch1-sec1"
          title: "简介"
          order: 1
          subsections:
            - id: "ch1-sec1-sub1"
              title: "项目背景"
              order: 1
              estimated_complexity: 2
              
            - id: "ch1-sec1-sub2"
              title: "核心功能"
              order: 2
              estimated_complexity: 3
              
    - id: "ch2"
      title: "架构设计"
      order: 2
      sections:
        - id: "ch2-sec1"
          title: "整体架构"
          order: 1
          subsections:
            - id: "ch2-sec1-sub1"
              title: "分层架构"
              order: 1
              estimated_complexity: 5
```

### 3. estimate_complexity

预估每个节点的复杂度权重。

**复杂度因素：**
- 代码行数
- 依赖数量
- 函数复杂度
- 业务重要性

**复杂度等级：**
- 1-2: 简单（概念性描述）
- 3-4: 中等（简要代码分析）
- 5-7: 复杂（详细代码解读）
- 8-10: 非常复杂（深入架构分析）

### 4. customize_for_branch

根据分支类型调整侧重点。

**分支类型影响：**

| 分支类型 | 文档侧重点 |
|---------|-----------|
| main/master | 完整架构文档 |
| develop | 开发规范、待完成功能 |
| feature/* | 变更说明、新功能设计 |
| release/* | 发布说明、部署指南 |

## 模板定义

### Web 服务模板

```yaml
template: web_service
chapters:
  - title: "项目概览"
    sections:
      - title: "简介"
      - title: "快速开始"
      - title: "目录结构"
      
  - title: "架构设计"
    sections:
      - title: "整体架构"
      - title: "模块划分"
      - title: "数据流"
      
  - title: "核心模块"
    sections:
      - title: "路由系统"
      - title: "中间件"
      - title: "服务层"
      - title: "数据访问层"
      
  - title: "API 文档"
    sections:
      - title: "认证机制"
      - title: "接口列表"
      
  - title: "开发规范"
    sections:
      - title: "代码规范"
      - title: "测试规范"
      - title: "部署流程"
```

### CLI 工具模板

```yaml
template: cli_tool
chapters:
  - title: "项目概览"
    sections:
      - title: "简介"
      - title: "安装"
      - title: "快速开始"
      
  - title: "命令参考"
    sections:
      - title: "全局选项"
      - title: "子命令"
      
  - title: "架构设计"
    sections:
      - title: "命令解析"
      - title: "执行流程"
      - title: "插件机制"
      
  - title: "开发指南"
    sections:
      - title: "添加新命令"
      - title: "配置文件"
```

### 算法库模板

```yaml
template: algorithm_lib
chapters:
  - title: "项目概览"
    sections:
      - title: "简介"
      - title: "安装"
      - title: "使用示例"
      
  - title: "核心算法"
    sections:
      - title: "算法 A"
      - title: "算法 B"
      
  - title: "API 文档"
    sections:
      - title: "主要接口"
      - title: "数据结构"
      
  - title: "实现细节"
    sections:
      - title: "性能优化"
      - title: "复杂度分析"
```

## 完整输出格式

```yaml
DocOutline:
  template: string
  chapters:
    - id: string
      title: string
      order: int
      sections:
        - id: string
          title: string
          order: int
          subsections:
            - id: string
              title: string
              order: int
              estimated_complexity: int
              
  complexity_weights:
    "ch1-sec1-sub1": 2
    "ch2-sec1-sub1": 5
    
  metadata:
    total_chapters: int
    total_sections: int
    estimated_pages: int
```

## 使用示例

```yaml
# 在 ArchitectAgent 中使用
skills:
  - doc-structure

task:
  name: 生成文档大纲
  steps:
    - action: doc-structure.generate
      input:
        repo_meta: "{{repo_meta}}"
        structure_analysis: "{{structure_analysis}}"
        dependency_analysis: "{{dependency_analysis}}"
        branch: "main"
      output: doc_outline
```

## 依赖

- structure-analysis（可选）
- dependency-mapping（可选）

## 最佳实践

1. 根据仓库类型选择合适的模板
2. 复杂度权重用于指导写作深度
3. 大型项目可以考虑只生成核心章节的大纲
4. 支持增量生成：先生成框架，再细化具体章节
