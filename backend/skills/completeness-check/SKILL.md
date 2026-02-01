---
name: completeness-check
description: 检查代码覆盖率。检查文档是否覆盖了所有关键代码点。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: quality-assurance
  priority: P2
---

# Completeness Check Skill

检查文档是否完整覆盖了代码的关键部分。

## 使用场景

- ReviewerAgent 验证文档完整性
- 确保没有遗漏重要函数
- 评估文档质量

## 功能能力

### 1. check_function_coverage

检查函数覆盖情况。

**输入：**
```yaml
draft_content: |
  ## 路由匹配
  
  `matchRoute` 函数负责匹配路由...
  
  [详细内容]

key_functions:
  - name: "matchRoute"
    file: "router.go"
    importance: "high"
    
  - name: "addRoute"
    file: "router.go"
    importance: "high"
    
  - name: "findHandler"
    file: "router.go"
    importance: "medium"
    
  - name: "parseParams"
    file: "params.go"
    importance: "medium"
```

**输出：**
```yaml
coverage_report:
  coverage_rate: 0.75  # 3/4 = 75%
  
  covered:
    - function: "matchRoute"
      location_in_doc: "第2段"
      depth: "详细"
      
  partially_covered:
    - function: "addRoute"
      location_in_doc: "第5段"
      depth: "简要提及"
      suggestion: "建议补充路由注册的具体逻辑"
      
  missing:
    - function: "findHandler"
      importance: "medium"
      suggestion: "添加对处理器查找逻辑的说明"
      
    - function: "parseParams"
      importance: "medium"
      suggestion: "参数解析是匹配的关键步骤，建议补充"
      
  recommendations:
    - priority: "high"
      action: "补充 addRoute 的详细说明"
      
    - priority: "medium"
      action: "添加 findHandler 和 parseParams 的说明"
```

### 2. identify_missing_explanations

识别缺失解释的部分。

**分析维度：**
- 关键算法未解释
- 错误处理未覆盖
- 边界情况未说明
- 性能特性未提及

**输出：**
```yaml
missing_explanations:
  - type: "algorithm"
    description: "前缀树遍历算法的时间复杂度"
    location_in_code: "router.go:67-89"
    importance: "high"
    suggestion: "添加时间复杂度分析和优化策略说明"
    
  - type: "error_handling"
    description: "路由未匹配时的错误处理"
    location_in_code: "router.go:95"
    importance: "medium"
    suggestion: "说明返回 nil 时的默认行为"
    
  - type: "edge_case"
    description: "路径参数冲突处理"
    location_in_code: "router.go:112-120"
    importance: "medium"
    suggestion: "添加参数冲突的示例和处理逻辑"
```

### 3. verify_all_paths_covered

验证所有代码路径都有解释。

**分析内容：**
- 条件分支（if/else/switch）
- 循环处理
- 错误返回路径
- 并发路径

**输出：**
```yaml
path_coverage:
  total_paths: 8
  covered_paths: 6
  coverage_rate: 0.75
  
  uncovered_paths:
    - path: "staticChild 为 nil 且 paramChild 为 nil 时返回 nil"
      location: "router.go:105"
      description: "未匹配到任何路由的情况"
      
    - path: "并发访问时的竞态条件处理"
      location: "router.go:45"
      description: "使用了读写锁保护"
```

## 完整输出格式

```yaml
CompletenessReport:
  overall_coverage: float
  function_coverage:
    total: int
    covered: int
    partial: int
    missing: int
  
  missing_points: array
  recommendations: array
  
  severity: "high" | "medium" | "low"
  ready_for_publish: boolean
```

## 检查清单

### 函数级别检查

- [ ] 所有重要函数都有提及
- [ ] 核心函数有详细解释
- [ ] 函数参数有说明
- [ ] 返回值有说明

### 逻辑级别检查

- [ ] 主要执行流程已覆盖
- [ ] 错误处理逻辑已说明
- [ ] 边界情况有提及
- [ ] 性能特性有说明

### 代码引用检查

- [ ] 关键代码片段已引用
- [ ] 代码位置标注清晰
- [ ] 代码和说明对应正确

## 使用示例

```yaml
# 在 ReviewerAgent 中使用
skills:
  - completeness-check

task:
  name: 检查完整性
  steps:
    - action: completeness-check.verify
      input:
        draft_content: "{{section_draft}}"
        key_functions: "{{title_context.key_functions}}"
      output: completeness_report
```

## 依赖

- filesystem.read
- code.parse_ast
- code.extract_functions

## 最佳实践

1. 高重要性函数必须详细覆盖
2. 中等重要性函数至少简要提及
3. 路径覆盖关注主流程和错误流程
4. 生成具体的改进建议而非仅指出问题
