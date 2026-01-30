# 007-典型仓库解读Agent系统-设计

## 1. 设计目标

基于需求文档 `docs/需求/典型仓库解读流程分析.md`，设计一套完整的 Agent、Skill、Tool 协作架构，支持 openDeepWiki 的典型仓库解读工作流。

## 2. 核心架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         OrchestratorAgent (协调者)                           │
│                    全局协调、任务调度、状态管理                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
       ┌───────────────────────────────┼───────────────────────────────┐
       ▼                               ▼                               ▼
┌──────────────┐            ┌──────────────────┐            ┌──────────────┐
│RepoInitializer│           │   ArchitectAgent  │            │ ExplorerAgent │
│ 仓库初始化    │            │    文档架构师      │            │   代码探索者   │
└──────────────┘            └──────────────────┘            └──────────────┘
       │                               │                               │
       ▼                               ▼                               ▼
┌──────────────┐            ┌──────────────────┐            ┌──────────────┐
│ PlannerAgent  │            │    WriterAgent   │            │ ReviewerAgent │
│  内容规划师   │            │     技术作者      │            │   质量审查员   │
└──────────────┘            └──────────────────┘            └──────────────┘
                                       │                               │
                                       ▼                               ▼
                            ┌──────────────────┐            ┌──────────────┐
                            │    EditorAgent   │            │   QAAgent    │
                            │      编辑        │            │   问答助手    │
                            └──────────────────┘            └──────────────┘
```

## 3. Agent 定义

### 3.1 OrchestratorAgent - 项目总协调

```yaml
name: orchestrator-agent
role: 项目总协调
system_prompt: |
  你是 openDeepWiki 的总控，负责管理文档生成全流程。
  你的职责：
  1. 维护全局上下文 GlobalContext，包括已完成的章节摘要、关键概念定义表
  2. 协调各 Agent 之间的任务流转
  3. 处理章节间依赖关系，决定阻塞等待/先写占位符/调整顺序
  4. 监控各 Agent 工作进度，防止重复处理
  5. 处理冲突（如两个章节引用了同一代码文件但解释矛盾）
  
  工作原则：
  - 只负责协调，不直接操作工具
  - 确保整体文档的一致性和生成效率
  - 当发现严重问题无法自动解决时，标记为"需人工确认"

skills:
  - dependency-management
  - task-scheduling
  - state-management
  - context-management

tools: []  # 主要协调其他 Agent，不直接操作工具

memory:
  type: long_term
  scope: global
  data:
    - global_context
    - project_state
    - chapter_summaries
    - concept_definitions
```

### 3.2 RepoInitializer - 仓库初始化专员

```yaml
name: repo-initializer
role: 仓库初始化专员
system_prompt: |
  你负责拉取代码并做初步分析，建立对仓库的基础认知。
  
  工作流程：
  1. 调用 GitTool 克隆仓库到工作目录
  2. 读取根目录文件（README/package.json/go.mod等）
  3. 统计代码文件类型分布
  4. 识别技术栈（Python项目/Go微服务/React前端等）
  
  输出要求：
  - 生成 RepoMeta 对象，包含：type, languages, entry_files, size, framework
  - 支持指定分支，根据分支类型调整分析侧重点

skills:
  - repo-detection
  - structure-analysis
  - dependency-mapping

tools:
  - git.clone
  - filesystem.ls
  - filesystem.read
  - code.parse_ast

memory:
  type: short_term
  scope: current_repo
  data:
    - repo_meta
```

### 3.3 ArchitectAgent - 文档架构师

```yaml
name: architect-agent
role: 文档架构师
system_prompt: |
  你设计文档的整体结构，不做细节撰写。
  
  工作流程：
  1. 获取文件树，识别模块边界（有__init__.py的文件夹/go package等）
  2. 分析依赖关系，找出核心模块 vs 辅助模块
  3. 结合仓库类型套用模板（Web项目：路由-控制器-模型；CLI工具：命令-参数-处理逻辑）
  4. 运用 DocStructureSkill 生成三级大纲：
     - 一级：核心概念/架构设计/使用指南/开发规范
     - 二级：各章节（如"路由系统"）
     - 三级：具体标题（如"动态路由匹配机制"）
  
  输出要求：
  - 产出 DocOutline：JSON结构化的三级目录
  - 包含每个节点的预估复杂度权重
  - 内置模板库，根据仓库类型自动选择

skills:
  - doc-structure
  - hierarchy-mapping
  - repo-detection

tools:
  - search.semantic
  - filesystem.grep
  - code.get_file_tree

memory:
  type: short_term
  scope: outline_generation
  data:
    - doc_outline
    - repo_meta
```

### 3.4 ExplorerAgent - 代码探索者

```yaml
name: explorer-agent
role: 代码探索者
system_prompt: |
  你深入代码库，为特定主题找到相关代码证据。
  
  工作流程：
  1. 接受任务：针对特定标题（如"动态路由匹配机制"）
  2. 调用 SemanticSearchTool 用标题语义搜索相关代码片段
  3. 用 GrepTool 搜索 keywords（如"router"、"match"、"dynamic"）
  4. 解析关键文件的 AST，提取函数/类定义
  5. Trace 依赖图，找出上下游关联文件
  6. 判断：哪些文件是核心(primary)，哪些是参考(secondary)
  
  输出要求：
  - 产出 TitleContext：{title, primary_files[], secondary_files[], key_functions[], data_flow_hints}
  - 缓存代码分析结果，避免重复解析

skills:
  - code-relevance
  - context-extraction
  - dependency-mapping

tools:
  - search.semantic
  - search.symbol
  - code.get_call_graph
  - code.extract_functions
  - code.parse_ast

memory:
  type: medium_term
  scope: current_title
  data:
    - title_context
    - cached_analysis
```

### 3.5 PlannerAgent - 内容规划师

```yaml
name: planner-agent
role: 内容规划师
system_prompt: |
  你将写作目标拆解为可执行的小节。
  
  工作流程：
  1. 读取 TitleContext 中的代码摘要
  2. 分析代码复杂度：简单函数直接说明，复杂机制拆分步骤
  3. 应用写作模板：
     - 概念速览（What）
     - 代码位置（Where）
     - 执行流程（How）- 可再细分：初始化→匹配→返回
     - 设计意图（Why）
     - 使用示例（Example）
     - 边界情况（Edge Cases）
  4. 生成小节标题列表，标记每个小节需要哪些代码证据
  
  输出要求：
  - 产出 SectionPlan：[{section_name, code_refs[], writing_goal, estimated_length}]

skills:
  - article-structuring
  - technical-writing

tools:
  - code.calculate_complexity

memory:
  type: short_term
  scope: section_planning
  data:
    - section_plan
```

### 3.6 WriterAgent - 技术作者

```yaml
name: writer-agent
role: 技术作者
system_prompt: |
  你将代码解释为人类可读的技术文档。
  
  工作流程：
  1. 读取 SectionPlan 中的第一个小节
  2. 调用 CodeParserTool 提取相关代码片段（自动截断关键部分）
  3. 使用 CodeExplanationSkill：
     - 对代码进行逐行/逐块注释
     - 识别设计模式（工厂/观察者/策略等）
     - 生成 Mermaid 流程图
  4. 使用 NarrativeFlowSkill 组织语言：
     - 先给结论，再给细节
     - 用比喻解释抽象概念
     - 保持技术准确性
  5. 若需要示例，调用 ExampleGenerationSkill 构造最小可运行示例
  
  输出要求：
  - 产出 SectionDraft：Markdown 格式的完整小节内容
  - 包含代码块、图表、说明文字

skills:
  - code-explanation
  - narrative-flow
  - example-generation
  - diagram-description

tools:
  - code.parse_ast
  - code.get_snippet
  - generation.llm_generate
  - generation.generate_mermaid
  - filesystem.read

memory:
  type: short_term
  scope: current_section
  data:
    - section_draft
```

### 3.7 ReviewerAgent - 质量审查员

```yaml
name: reviewer-agent
role: 质量审查员
system_prompt: |
  你检查文档的完整性、准确性和一致性。
  
  工作流程：
  1. 代码覆盖检查：比对 SectionDraft 中提到的代码 vs TitleContext 中标记的关键函数
  2. 逻辑一致性：验证前文提到的概念在后文有解释，示例代码与描述一致
  3. 交叉引用：检查是否引用了其他章节的内容，确保链接有效
  4. 风格检查：确保术语统一（如别前面叫"路由"后面叫"Router"）
  5. 发现严重技术理解错误时，标记为"需人工确认"而非自动修复
  
  输出要求：
  - 产出 ReviewReport：{missing_points[], inconsistencies[], suggestions[]}
  - 或直接输出修复后的 SectionFinal

skills:
  - completeness-check
  - consistency-check
  - technical-accuracy
  - style-check

tools:
  - quality.check_links
  - quality.plagiarism_check
  - filesystem.read

memory:
  type: short_term
  scope: review_task
  data:
    - review_report
```

### 3.8 EditorAgent - 编辑

```yaml
name: editor-agent
role: 编辑
system_prompt: |
  你组装最终文档，优化阅读体验。
  
  工作流程：
  1. 收集该章节下所有 SectionFinal
  2. 检查小节间逻辑流，添加过渡句
  3. 生成章节摘要（TL;DR）
  4. 统一代码块样式、图表编号
  5. 添加"相关章节"链接
  
  输出要求：
  - 产出 ChapterDocument：完整的可发布章节

skills:
  - content-assembly
  - transition-optimization
  - summary-generation

tools:
  - quality.check_links

memory:
  type: medium_term
  scope: chapter_assembly
  data:
    - chapter_document
```

### 3.9 QAAgent - 问答助手（特殊Agent）

```yaml
name: qa-agent
role: 问答助手
system_prompt: |
  你为用户提供精准的技术问答服务。
  
  工作流程：
  1. 接收用户问题
  2. 查询已生成文档+原始代码
  3. 给出精准回答
  4. 可选地将 QA 沉淀为新章节
  
  能力：
  - 语义检索已生成文档
  - 代码片段定位
  - 综合多个信息源生成回答

skills:
  - retrieval
  - synthesis
  - code-explanation

tools:
  - search.semantic
  - generation.llm_generate
  - filesystem.read

memory:
  type: long_term
  scope: qa_session
  data:
    - qa_history
```

## 4. Skill 定义

### 4.1 仓库理解类 (RepositoryUnderstanding)

#### Skill: repo-detection

```yaml
name: repo-detection
description: 识别技术栈和项目类型
version: "1.0"
category: repository-understanding

capabilities:
  - detect_language_distribution: 统计代码文件类型分布
  - detect_framework: 识别使用的框架（Django, Spring, React等）
  - detect_project_type: 识别项目类型（Web服务/CLI工具/算法库等）
  - detect_entry_points: 识别入口文件

inputs:
  - repo_path: 仓库本地路径
  
outputs:
  - repo_meta:
      type: object
      properties:
        type: string
        languages: map<string, int>
        framework: string
        entry_files: string[]
        size: int
```

#### Skill: structure-analysis

```yaml
name: structure-analysis
description: 分析目录结构和模块边界
version: "1.0"
category: repository-understanding

capabilities:
  - analyze_directory_structure: 分析目录层级和命名规范
  - identify_module_boundaries: 识别模块边界（package等）
  - analyze_naming_conventions: 分析命名规范一致性
  - detect_architecture_pattern: 检测架构模式（MVC/分层/微服务等）

inputs:
  - repo_path: 仓库本地路径
  - repo_meta: 仓库元数据
  
outputs:
  - structure_analysis:
      type: object
      properties:
        modules: array
        boundaries: array
        patterns: array
```

#### Skill: dependency-mapping

```yaml
name: dependency-mapping
description: 映射模块间依赖关系
version: "1.0"
category: repository-understanding

capabilities:
  - build_dependency_graph: 构建模块依赖图
  - identify_core_modules: 识别核心模块 vs 辅助模块
  - detect_circular_dependencies: 检测循环依赖
  - analyze_import_patterns: 分析导入模式

inputs:
  - repo_path: 仓库本地路径
  - language: 主要编程语言
  
outputs:
  - dependency_graph:
      type: object
      properties:
        nodes: array
        edges: array
        core_modules: array
```

### 4.2 内容规划类 (ContentPlanning)

#### Skill: doc-structure

```yaml
name: doc-structure
description: 根据项目类型生成文档大纲模板
version: "1.0"
category: content-planning

capabilities:
  - select_template: 根据仓库类型选择模板
  - generate_outline: 生成三级文档大纲
  - estimate_complexity: 预估每个节点的复杂度权重
  - customize_for_branch: 根据分支类型调整侧重点

templates:
  - web_service: Web服务项目模板
  - cli_tool: CLI工具项目模板
  - algorithm_lib: 算法库项目模板
  - frontend_app: 前端应用模板
  - microservice: 微服务架构模板

inputs:
  - repo_meta: 仓库元数据
  - structure_analysis: 结构分析结果
  - branch: 分支名称（可选）
  
outputs:
  - doc_outline:
      type: object
      properties:
        chapters: array
        complexity_weights: map<string, int>
```

#### Skill: hierarchy-mapping

```yaml
name: hierarchy-mapping
description: 将代码结构映射为文档层级
version: "1.0"
category: content-planning

capabilities:
  - map_code_to_doc: 将代码结构映射到文档结构
  - organize_by_concern: 按关注点组织（而非仅按目录）
  - balance_depth: 平衡文档深度和阅读体验

inputs:
  - structure_analysis: 结构分析结果
  - dependency_graph: 依赖关系图
  
outputs:
  - hierarchy_map:
      type: object
      properties:
        mappings: array
```

#### Skill: code-relevance

```yaml
name: code-relevance
description: 判断代码与写作目标的相关性
version: "1.0"
category: content-planning

capabilities:
  - score_relevance: 为代码片段与主题的相关性打分
  - classify_importance: 分类重要性（primary/secondary/reference）
  - rank_by_relevance: 按相关性排序

inputs:
  - topic: 写作主题
  - code_snippets: 代码片段列表
  
outputs:
  - relevance_result:
      type: object
      properties:
        primary_files: array
        secondary_files: array
        scores: map<string, float>
```

### 4.3 写作类 (Writing)

#### Skill: code-explanation

```yaml
name: code-explanation
description: 将代码逻辑转化为自然语言
version: "1.0"
category: writing

capabilities:
  - explain_function: 解释函数逻辑
  - explain_class: 解释类设计
  - explain_data_flow: 解释数据流
  - identify_patterns: 识别设计模式
  - generate_inline_comments: 生成行内注释

inputs:
  - code_snippet: 代码片段
  - context: 上下文信息
  
outputs:
  - explanation: string
  - patterns_detected: array
  - key_points: array
```

#### Skill: narrative-flow

```yaml
name: narrative-flow
description: 组织技术叙事结构
version: "1.0"
category: writing

capabilities:
  - organize_structure: 组织"What-Where-How-Why"结构
  - add_transitions: 添加过渡段落
  - maintain_coherence: 保持叙述连贯性
  - apply_inverted_pyramid: 应用倒金字塔结构（先结论后细节）

inputs:
  - content_sections: 内容段落列表
  - target_audience: 目标读者
  
outputs:
  - structured_content: string
```

#### Skill: example-generation

```yaml
name: example-generation
description: 生成使用示例
version: "1.0"
category: writing

capabilities:
  - generate_minimal_example: 生成最小可运行示例
  - generate_usage_scenario: 生成使用场景示例
  - generate_edge_case_example: 生成边界情况示例

inputs:
  - code_context: 代码上下文
  - example_type: 示例类型
  
outputs:
  - example_code: string
  - explanation: string
```

#### Skill: diagram-description

```yaml
name: diagram-description
description: 生成图表描述文字（Mermaid等）
version: "1.0"
category: writing

capabilities:
  - generate_flowchart: 生成流程图描述
  - generate_sequence_diagram: 生成时序图描述
  - generate_class_diagram: 生成类图描述
  - generate_architecture_diagram: 生成架构图描述

inputs:
  - code_snippet: 代码片段
  - diagram_type: 图表类型
  
outputs:
  - mermaid_code: string
  - description: string
```

### 4.4 质量保障类 (QualityAssurance)

#### Skill: completeness-check

```yaml
name: completeness-check
description: 检查代码覆盖率
version: "1.0"
category: quality-assurance

capabilities:
  - check_function_coverage: 检查函数覆盖情况
  - identify_missing_explanations: 识别缺失解释的部分
  - verify_all_paths_covered: 验证所有代码路径都有解释

inputs:
  - draft_content: 草稿内容
  - key_functions: 关键函数列表
  
outputs:
  - coverage_report:
      type: object
      properties:
        coverage_rate: float
        missing: array
```

#### Skill: consistency-check

```yaml
name: consistency-check
description: 检查术语和逻辑一致性
version: "1.0"
category: quality-assurance

capabilities:
  - check_terminology: 检查术语一致性
  - check_logic_flow: 检查逻辑流程一致性
  - check_cross_references: 检查交叉引用有效性

inputs:
  - document_content: 文档内容
  - terminology_dict: 术语词典（可选）
  
outputs:
  - consistency_report:
      type: object
      properties:
        inconsistencies: array
        suggestions: array
```

#### Skill: technical-accuracy

```yaml
name: technical-accuracy
description: 验证技术描述的正确性
version: "1.0"
category: quality-assurance

capabilities:
  - verify_code_description: 验证代码描述准确性
  - check_api_usage: 检查API使用描述
  - validate_algorithms: 验证算法描述

inputs:
  - description: 描述文本
  - source_code: 源代码
  
outputs:
  - accuracy_report:
      type: object
      properties:
        is_accurate: boolean
        issues: array
```

#### Skill: style-check

```yaml
name: style-check
description: 检查写作风格和格式规范
version: "1.0"
category: quality-assurance

capabilities:
  - check_markdown_format: 检查Markdown格式
  - check_code_block_style: 检查代码块样式
  - check_heading_hierarchy: 检查标题层级

inputs:
  - document_content: 文档内容
  
outputs:
  - style_report:
      type: object
      properties:
        issues: array
        formatted_version: string
```

### 4.5 协调类 (Coordination)

#### Skill: dependency-management

```yaml
name: dependency-management
description: 解决章节间依赖
version: "1.0"
category: coordination

capabilities:
  - resolve_dependencies: 解析章节依赖关系
  - detect_conflicts: 检测章节间冲突
  - suggest_resolution: 建议解决方案

inputs:
  - chapter_plans: 章节计划列表
  
outputs:
  - resolution_plan:
      type: object
      properties:
        execution_order: array
        dependencies: map
```

#### Skill: task-scheduling

```yaml
name: task-scheduling
description: 任务调度
version: "1.0"
category: coordination

capabilities:
  - schedule_tasks: 调度任务执行顺序
  - optimize_parallelism: 优化并行执行
  - handle_failures: 处理失败重试

inputs:
  - tasks: 任务列表
  - dependencies: 依赖关系
  
outputs:
  - schedule:
      type: object
      properties:
        order: array
        parallel_groups: array
```

#### Skill: state-management

```yaml
name: state-management
description: 维护全局状态
version: "1.0"
category: coordination

capabilities:
  - track_progress: 跟踪进度
  - persist_state: 持久化状态
  - restore_state: 恢复状态

inputs:
  - operation: 操作类型
  - state_data: 状态数据
  
outputs:
  - current_state: object
```

#### Skill: context-management

```yaml
name: context-management
description: 维护全局上下文和记忆
version: "1.0"
category: coordination

capabilities:
  - update_global_context: 更新全局上下文
  - manage_concepts: 管理概念定义表
  - share_context: 在Agent间共享上下文

inputs:
  - context_update: 上下文更新
  
outputs:
  - global_context: object
```

#### Skill: content-assembly

```yaml
name: content-assembly
description: 内容组装
version: "1.0"
category: coordination

capabilities:
  - assemble_sections: 组装各小节为完整章节
  - generate_toc: 生成目录
  - add_navigation: 添加导航链接

inputs:
  - sections: 小节列表
  - chapter_metadata: 章节元数据
  
outputs:
  - chapter_document: string
```

#### Skill: transition-optimization

```yaml
name: transition-optimization
description: 过渡优化
version: "1.0"
category: coordination

capabilities:
  - add_transition_paragraphs: 添加过渡段落
  - smooth_flow: 平滑段落衔接

inputs:
  - sections: 小节列表
  
outputs:
  - optimized_content: string
```

#### Skill: summary-generation

```yaml
name: summary-generation
description: 摘要生成
version: "1.0"
category: coordination

capabilities:
  - generate_chapter_summary: 生成章节摘要
  - generate_tldr: 生成TL;DR

inputs:
  - chapter_content: 章节内容
  
outputs:
  - summary: string
```

## 5. MCP Tools 定义

### 5.1 GitTools

```yaml
namespace: git
description: Git 操作工具

tools:
  - name: clone
    description: 克隆指定分支
    parameters:
      repo_url: string
      branch: string (optional)
      target_dir: string
    returns:
      success: boolean
      path: string

  - name: diff
    description: 获取变更差异
    parameters:
      commit_hash: string
      file_path: string (optional)
    returns:
      diff_content: string

  - name: log
    description: 获取文件提交历史
    parameters:
      file_path: string
      limit: int (optional, default=10)
    returns:
      commits: array
```

### 5.2 FileSystemTools

```yaml
namespace: filesystem
description: 文件系统操作工具

tools:
  - name: ls
    description: 列出目录结构
    parameters:
      dir: string
      recursive: boolean (optional, default=false)
    returns:
      entries: array

  - name: read
    description: 读取文件内容
    parameters:
      file_path: string
      offset: int (optional)
      limit: int (optional)
    returns:
      content: string

  - name: grep
    description: 正则搜索
    parameters:
      pattern: string
      path: string
      recursive: boolean (optional, default=true)
    returns:
      matches: array

  - name: stat
    description: 文件元信息
    parameters:
      file_path: string
    returns:
      size: int
      modified: timestamp
      is_dir: boolean
```

### 5.3 CodeAnalysisTools

```yaml
namespace: code
description: 代码分析工具

tools:
  - name: parse_ast
    description: 生成 AST
    parameters:
      file_path: string
    returns:
      ast: object

  - name: extract_functions
    description: 提取函数列表
    parameters:
      file_path: string
    returns:
      functions: array

  - name: get_call_graph
    description: 生成调用图
    parameters:
      entry_point: string
    returns:
      graph: object

  - name: calculate_complexity
    description: 计算圈复杂度
    parameters:
      file_path: string
    returns:
      complexity: int
      details: object

  - name: get_file_tree
    description: 获取文件树
    parameters:
      repo_path: string
    returns:
      tree: object

  - name: get_snippet
    description: 获取代码片段
    parameters:
      file_path: string
      line_start: int
      line_end: int
    returns:
      snippet: string
```

### 5.4 SearchTools

```yaml
namespace: search
description: 搜索工具

tools:
  - name: semantic
    description: 基于嵌入的语义搜索
    parameters:
      query: string
      repo_path: string
      top_k: int (optional, default=10)
    returns:
      results: array

  - name: symbol
    description: 精确符号搜索
    parameters:
      symbol_name: string
      repo_path: string
    returns:
      locations: array
```

### 5.5 GenerationTools

```yaml
namespace: generation
description: 生成工具

tools:
  - name: llm_generate
    description: 调用 LLM 生成内容
    parameters:
      prompt: string
      context: object (optional)
      model: string (optional)
    returns:
      content: string

  - name: generate_mermaid
    description: 生成 Mermaid 流程图
    parameters:
      code_snippet: string
      diagram_type: string (optional, default="flowchart")
    returns:
      mermaid_code: string

  - name: generate_diagram
    description: 生成架构图描述
    parameters:
      description: string
      diagram_type: string
    returns:
      diagram_code: string
```

### 5.6 QualityTools

```yaml
namespace: quality
description: 质量检查工具

tools:
  - name: check_links
    description: 检查内部链接有效性
    parameters:
      doc_content: string
      base_path: string
    returns:
      broken_links: array

  - name: plagiarism_check
    description: 检查与代码的重复度
    parameters:
      text: string
      code_files: array
    returns:
      similarity_score: float
```

## 6. 数据模型

### 6.1 RepoMeta

```typescript
interface RepoMeta {
  type: string;              // 项目类型
  languages: Map<string, number>;  // 语言分布
  framework: string;         // 主要框架
  entry_files: string[];     // 入口文件
  size: number;              // 代码规模（行数）
}
```

### 6.2 DocOutline

```typescript
interface DocOutline {
  chapters: Chapter[];
  complexity_weights: Map<string, number>;
}

interface Chapter {
  id: string;
  title: string;
  sections: Section[];
}

interface Section {
  id: string;
  title: string;
  subsections: SubSection[];
}

interface SubSection {
  id: string;
  title: string;
  estimated_complexity: number;
}
```

### 6.3 TitleContext

```typescript
interface TitleContext {
  title: string;
  primary_files: string[];
  secondary_files: string[];
  key_functions: FunctionInfo[];
  data_flow_hints: DataFlowHint[];
}

interface FunctionInfo {
  name: string;
  file: string;
  line: number;
  signature: string;
}

interface DataFlowHint {
  source: string;
  target: string;
  description: string;
}
```

### 6.4 SectionPlan

```typescript
interface SectionPlan {
  sections: PlannedSection[];
}

interface PlannedSection {
  section_name: string;
  code_refs: CodeRef[];
  writing_goal: string;
  estimated_length: number;
}

interface CodeRef {
  file: string;
  lines: [number, number];
  type: 'primary' | 'secondary';
}
```

### 6.5 ReviewReport

```typescript
interface ReviewReport {
  missing_points: string[];
  inconsistencies: Inconsistency[];
  suggestions: Suggestion[];
  needs_human_review: boolean;
}

interface Inconsistency {
  type: 'terminology' | 'logic' | 'reference';
  description: string;
  locations: string[];
}

interface Suggestion {
  priority: 'high' | 'medium' | 'low';
  description: string;
  action: string;
}
```

## 7. 协作流程

```
OrchestratorAgent (启动)
    │
    ▼
调用 RepoInitializer
    │
    ▼
产出 RepoMeta
    │
    ▼
调用 ArchitectAgent
    │
    ▼
产出 DocOutline
    │
    ├───────────────────────────────┐
    ▼                               ▼
调用 ExplorerAgent (章节1)      调用 ExplorerAgent (章节2)
    │                               │
    ▼                               ▼
产出 TitleContext                产出 TitleContext
    │                               │
    ▼                               ▼
调用 PlannerAgent                调用 PlannerAgent
    │                               │
    ▼                               ▼
产出 SectionPlan                 产出 SectionPlan
    │                               │
    ▼                               ▼
调用 WriterAgent (迭代N次)       调用 WriterAgent (迭代N次)
    │                               │
    ▼                               ▼
产出 SectionDraft                产出 SectionDraft
    │                               │
    ▼                               ▼
调用 ReviewerAgent               调用 ReviewerAgent
    │                               │
    ▼                               ▼
返回修改建议或 SectionFinal      返回修改建议或 SectionFinal
    │                               │
    └───────────────┬───────────────┘
                    ▼
            调用 EditorAgent
                    │
                    ▼
            产出 ChapterDocument
                    │
                    ▼
            返回 OrchestratorAgent
                    │
                    ▼
            合并到全局文档
```

## 8. 扩展性设计

1. **新增仓库类型**：只需调整 Skills（添加新的模板）
2. **新增输出格式**：只需调整 WriterAgent
3. **新增分析维度**：可添加新的 Skill 类别
4. **新增工具**：通过 MCP 机制动态加载

## 9. 关键设计决策

1. **分支管理**：RepoInitializer 支持指定分支，ArchitectAgent 根据分支类型调整文档侧重点
2. **缓存策略**：ExplorerAgent 的代码分析结果缓存（存储向量嵌入）
3. **人机协同**：ReviewerAgent 发现严重技术理解错误时，标记为"需人工确认"
4. **增量生成**：支持"只解读 src/core 目录"或"只生成架构相关章节"的局部生成
5. **模板化**：ArchitectAgent 内置模板库，根据仓库类型自动选择

## 10. 实现优先级

### P0 - 核心功能
- [ ] OrchestratorAgent
- [ ] RepoInitializer
- [ ] ArchitectAgent
- [ ] repo-detection Skill
- [ ] structure-analysis Skill
- [ ] doc-structure Skill

### P1 - 文档生成
- [ ] ExplorerAgent
- [ ] PlannerAgent
- [ ] WriterAgent
- [ ] code-relevance Skill
- [ ] code-explanation Skill
- [ ] narrative-flow Skill

### P2 - 质量控制
- [ ] ReviewerAgent
- [ ] EditorAgent
- [ ] completeness-check Skill
- [ ] consistency-check Skill

### P3 - 高级功能
- [ ] QAAgent
- [ ] 所有 Coordination Skills
- [ ] 语义搜索 Tool
