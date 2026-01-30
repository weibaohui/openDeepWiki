---
name: technical-accuracy
description: 验证技术描述的正确性。检查代码描述、API使用、算法描述是否准确。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: quality-assurance
  priority: P2
---

# Technical Accuracy Skill

验证文档中的技术描述是否准确。

## 使用场景

- ReviewerAgent 验证技术正确性
- 防止错误的技术信息传播
- 标记需要人工确认的问题

## 功能能力

### 1. verify_code_description

验证代码描述准确性。

**输入：**
```yaml
description: |
  `matchRoute` 函数使用前缀树实现 O(n) 时间复杂度的路由匹配，
  其中 n 是路径段的数量。

source_code: |
  func (r *Router) matchRoute(path string) (*Route, Params) {
      segments := strings.Split(path, "/")
      node := r.root
      
      for _, seg := range segments {
          if child := node.staticChild(seg); child != nil {
              node = child
              continue
          }
          if child := node.paramChild(); child != nil {
              params.Add(child.paramName, seg)
              node = child
              continue
          }
          return nil, nil
      }
      
      return node.route, params
  }
```

**输出：**
```yaml
accuracy_report:
  is_accurate: true
  confidence: 0.9
  
  verified_claims:
    - claim: "使用前缀树"
      accurate: true
      evidence: "代码中使用了 node 树结构进行匹配"
      
    - claim: "O(n) 时间复杂度"
      accurate: true
      evidence: "单次遍历路径段，每次操作 O(1)"
      notes: "n 为路径段数，实际为 O(m)，m 为路径深度"
      
  warnings:
    - type: "oversimplification"
      description: "未提及最坏情况（大量通配符匹配）"
      suggestion: "添加关于通配符匹配的复杂度说明"
```

### 2. check_api_usage

检查 API 使用描述。

**输入：**
```yaml
description: |
  使用 `r.GET("/user/:id", handler)` 注册 GET 路由，
  其中 `:id` 是路径参数，可通过 `c.Param("id")` 获取。

api_definition:
  package: "github.com/gin-gonic/gin"
  methods:
    - name: "GET"
      signature: "func (r *RouterGroup) GET(path string, handlers ...HandlerFunc)"
    - name: "Param"
      signature: "func (c *Context) Param(key string) string"
```

**输出：**
```yaml
api_report:
  accurate: true
  
  checks:
    - element: "GET 方法"
      signature_match: true
      parameter_count_match: true
      
    - element: "Param 方法"
      signature_match: true
      usage_correct: true
      
  suggestions:
    - "可以补充说明 handlers 支持多个中间件"
    - "Param 返回 string，需要手动转换为其他类型"
```

### 3. validate_algorithms

验证算法描述。

**验证项：**
- 时间复杂度正确性
- 空间复杂度正确性
- 算法步骤正确性
- 边界条件处理

**输出：**
```yaml
algorithm_report:
  algorithm_name: "前缀树路由匹配"
  
  time_complexity:
    claimed: "O(n)"
    actual: "O(m)"
    explanation: "m 为路径深度（段数），通常 m << n（总路由数）"
    accurate: true
    
  space_complexity:
    claimed: "未提及"
    actual: "O(k * m)"
    explanation: "k 为路由数，m 为平均路径深度"
    suggestion: "添加空间复杂度说明"
    
  steps:
    - step: 1
      description: "分割路径"
      correct: true
      
    - step: 2
      description: "遍历前缀树"
      correct: true
      
    - step: 3
      description: "返回匹配结果"
      correct: true
      note: "应说明返回 nil 表示未匹配"
      
  edge_cases:
    - case: "空路径"
      handled: true
      documented: false
      suggestion: "添加空路径处理说明"
      
    - case: "路径参数冲突"
      handled: true
      documented: true
```

## 完整输出格式

```yaml
TechnicalAccuracyReport:
  overall_accuracy: float
  confidence: float
  needs_human_review: boolean
  
  code_verification:
    accurate: boolean
    claims: array
    warnings: array
    
  api_verification:
    accurate: boolean
    checks: array
    
  algorithm_verification:
    time_complexity: object
    space_complexity: object
    steps: array
    edge_cases: array
    
  critical_issues: array
```

## 严重程度分级

| 级别 | 说明 | 处理方式 |
|-----|------|---------|
| Critical | 技术事实错误 | 必须修复，否则标记为需人工确认 |
| Warning | 描述不精确或遗漏 | 建议补充或修正 |
| Info | 可以改进的建议 | 可选采纳 |

## 使用示例

```yaml
# 在 ReviewerAgent 中使用
skills:
  - technical-accuracy

task:
  name: 验证技术准确性
  steps:
    - action: technical-accuracy.validate
      input:
        description: "{{section_draft}}"
        source_code: "{{source_files}}"
        api_definitions: "{{api_docs}}"
      output: accuracy_report
```

## 依赖

- filesystem.read
- code.parse_ast
- code.calculate_complexity

## 最佳实践

1. 复杂的技术声明需要给出证据
2. 不确定的问题标记为需人工确认
3. 算法复杂度分析要考虑最坏情况
4. API 描述要与官方文档一致
5. 对于模糊的技术点，给出说明范围
