---
name: consistency-check
description: 检查术语和逻辑一致性。确保文档前后一致，交叉引用有效。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: quality-assurance
  priority: P2
---

# Consistency Check Skill

检查文档的一致性，包括术语、逻辑和交叉引用。

## 使用场景

- ReviewerAgent 验证文档一致性
- 确保术语统一
- 检查交叉引用有效性

## 功能能力

### 1. check_terminology

检查术语一致性。

**输入：**
```yaml
document_content: |
  ## 路由系统
  
  Router 负责管理所有的 Route。当请求到达时，
  路由系统会匹配对应的处理器。
  
  ## 处理器
  
  Handler 是处理 HTTP 请求的函数。每个路由对应一个 Handler。
  
terminology_dict:
  Router: ["路由器", "路由系统"]
  Route: ["路由"]
  Handler: ["处理器", "处理函数"]
  Middleware: ["中间件"]
```

**输出：**
```yaml
terminology_report:
  consistency_score: 0.85
  
  inconsistencies:
    - term: "Router"
      issue: "混用"
      occurrences:
        - location: "第1段"
          usage: "Router"
        - location: "第2段"
          usage: "路由系统"
      suggestion: "统一使用'路由器'或'Router'，避免混用"
      
    - term: "Handler"
      issue: "大小写不一致"
      occurrences:
        - location: "标题"
          usage: "处理器"
        - location: "正文"
          usage: "Handler"
      suggestion: "首次出现时标注'处理器（Handler）'，后续统一使用'处理器'"
      
  preferred_terms:
    Router: "路由器（Router）"
    Route: "路由"
    Handler: "处理器"
    Middleware: "中间件"
```

### 2. check_logic_flow

检查逻辑流程一致性。

**检查项：**
- 前文提到的概念后文有解释
- 示例代码与描述一致
- 时序描述符合实际代码
- 数字数据前后一致

**输出：**
```yaml
logic_report:
  issues:
    - type: "unexplained_concept"
      concept: "前缀树"
      first_mentioned: "第3段"
      explanation_location: null
      suggestion: "首次出现时应简要解释前缀树的概念"
      
    - type: "inconsistent_example"
      description: "文字描述使用 /user/123，示例代码使用 /post/456"
      locations: ["第5段", "代码示例"]
      suggestion: "统一使用相同的示例路径"
      
    - type: "incorrect_sequence"
      description: "描述的处理顺序与实际代码不符"
      doc_order: ["分割路径", "匹配路由", "提取参数"]
      actual_order: ["分割路径", "提取参数", "匹配路由"]
      suggestion: "修正处理顺序描述"
```

### 3. check_cross_references

检查交叉引用有效性。

**输入：**
```yaml
document_content: |
  路由匹配的具体实现详见[路由匹配算法](#路由匹配算法)。
  
  关于中间件的使用，请参考[中间件章节](../middleware/overview.md)。
  
  性能优化技巧见[性能优化指南](./performance.md#优化技巧)。

existing_documents:
  - "./algorithm.md"
  - "../middleware/overview.md"
  - "./performance.md"
```

**输出：**
```yaml
link_report:
  valid_links: 2
  broken_links: 1
  
  checks:
    - link: "[路由匹配算法](#路由匹配算法)"
      type: "anchor"
      target: "#路由匹配算法"
      valid: false
      suggestion: "当前文档中不存在此锚点，请改为正确的锚点或相对链接"
      
    - link: "[中间件章节](../middleware/overview.md)"
      type: "relative"
      target: "../middleware/overview.md"
      valid: true
      
    - link: "[性能优化指南](./performance.md#优化技巧)"
      type: "relative_with_anchor"
      target: "./performance.md"
      anchor: "#优化技巧"
      valid: true
      anchor_valid: "unknown"  # 需要打开文件确认
      
  suggestions:
    - "修复无效的内部链接"
    - "验证外部链接的可访问性"
```

## 完整输出格式

```yaml
ConsistencyReport:
  overall_score: float
  
  terminology:
    score: float
    inconsistencies: array
    preferred_terms: object
    
  logic:
    score: float
    issues: array
    
  cross_references:
    valid_count: int
    broken_count: int
    issues: array
    
  severity: "high" | "medium" | "low"
```

## 术语一致性规则

### 1. 首次定义规则

```markdown
<!-- 好的写法 -->
路由器（Router）是负责管理路由的核心组件。

<!-- 后续使用 -->
路由器根据请求路径查找对应的处理函数。

<!-- 不好的写法 -->
Router 是负责管理路由的核心组件。
...
路由系统根据请求路径查找对应的处理函数。  <!-- 未说明与 Router 的关系 -->
```

### 2. 中英文统一

| 推荐 | 避免 |
|-----|-----|
| 路由器（Router） | Router / 路由系统 / 路由管理器 |
| 处理器（Handler） | Handler / 处理函数 / 处理程序 |
| 中间件（Middleware） | Middleware / 中间层 |

### 3. 大小写统一

- 代码中的标识符保持原样（Router, matchRoute）
- 中文术语使用正常大小写
- 英文缩写统一大写（API, HTTP, URL）

## 使用示例

```yaml
# 在 ReviewerAgent 中使用
skills:
  - consistency-check

task:
  name: 检查一致性
  steps:
    - action: consistency-check.verify
      input:
        document_content: "{{section_draft}}"
        terminology_dict: "{{global_context.terminology}}"
        existing_documents: "{{doc_structure.all_files}}"
      output: consistency_report
```

## 依赖

- filesystem.ls
- quality.check_links

## 最佳实践

1. 维护全局术语词典
2. 首次出现专业术语时给出定义
3. 定期检查交叉引用
4. 多人协作时使用统一的术语表
