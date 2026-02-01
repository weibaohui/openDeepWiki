---
name: code-explanation
description: 将代码逻辑转化为自然语言。解释函数、类、数据流，识别设计模式。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: writing
  priority: P1
---

# Code Explanation Skill

将代码转化为人类可读的技术文档。

## 使用场景

- WriterAgent 解释代码逻辑
- 生成代码注释
- 识别设计模式

## 功能能力

### 1. explain_function

解释函数逻辑。

**输入：**
```yaml
function_info:
  name: "matchRoute"
  file: "router.go"
  signature: "func (r *Router) matchRoute(path string) (*Route, Params)"
  body: "..."
context:
  purpose: "路由匹配"
  related_functions: ["addRoute", "findRoute"]
```

**输出：**
```yaml
explanation: |
  `matchRoute` 函数负责将 HTTP 请求路径匹配到对应的路由处理器。
  
  **匹配流程：**
  1. 将路径按 `/` 分割为多个段
  2. 从路由树的根节点开始遍历
  3. 对于每个路径段，查找匹配的子节点
  4. 如果遇到动态参数（如 `:id`），提取参数值
  5. 到达叶子节点时返回匹配的路由和参数
  
  **关键逻辑：**
  - 静态路由优先于动态路由匹配
  - 支持通配符 `*` 匹配任意后续路径
  
key_points:
  - "使用前缀树（Trie）实现高效匹配"
  - "时间复杂度 O(n)，n 为路径段数"
  - "支持路径参数提取"
```

### 2. explain_class

解释类设计。

**输出：**
```yaml
class_explanation: |
  `Router` 类是整个路由系统的核心，负责管理路由注册和请求分发。
  
  **职责：**
  - 维护路由树数据结构
  - 提供路由注册接口
  - 处理请求匹配
  
  **协作关系：**
  - 与 `Route` 类：一对多关系，Router 包含多个 Route
  - 与 `Middleware` 接口：使用组合模式添加中间件
  
design_patterns:
  - "策略模式：不同路由匹配策略"
  - "责任链模式：中间件执行链"
```

### 3. explain_data_flow

解释数据流。

**输出：**
```yaml
data_flow_explanation: |
  **请求处理数据流：**
  
  1. HTTP 请求到达 `Server`
  2. `Server` 调用 `Router.matchRoute` 进行路由匹配
  3. 匹配成功后，创建 `Context` 对象封装请求信息
  4. 按顺序执行中间件链
  5. 调用最终的 `Handler` 处理请求
  6. `Handler` 返回响应数据
  7. 响应通过中间件链返回
  8. `Server` 发送 HTTP 响应

flow_steps:
  - step: 1
    component: "Server"
    action: "接收请求"
    data: "HTTP Request"
    
  - step: 2
    component: "Router"
    action: "路由匹配"
    data: "Route, Params"
```

### 4. identify_patterns

识别设计模式。

**支持的模式：**
- 创建型：工厂、单例、建造者、原型
- 结构型：适配器、装饰器、代理、外观、桥接、组合、享元
- 行为型：策略、观察者、责任链、命令、迭代器、中介者、备忘录、状态、模板方法、访问者

**输出：**
```yaml
design_patterns:
  - pattern: "Strategy"
    confidence: 0.9
    location: "router.go:45"
    description: "使用不同的匹配策略处理静态和动态路由"
    participants:
      - "RouteMatcher (策略接口)"
      - "StaticMatcher (具体策略)"
      - "DynamicMatcher (具体策略)"
      - "Router (上下文)"
      
  - pattern: "Chain of Responsibility"
    confidence: 0.85
    location: "middleware.go:23"
    description: "中间件链依次处理请求"
```

### 5. generate_inline_comments

生成行内注释。

**输出：**
```go
// matchRoute 将请求路径匹配到注册的路由
// 返回匹配的路由和路径参数
func (r *Router) matchRoute(path string) (*Route, Params) {
    // 按 '/' 分割路径，例如 "/user/123" → ["user", "123"]
    segments := splitPath(path)
    
    // 从根节点开始遍历前缀树
    node := r.root
    
    // 遍历每个路径段
    for _, seg := range segments {
        // 优先匹配静态节点
        if child := node.staticChild(seg); child != nil {
            node = child
            continue
        }
        
        // 其次匹配动态参数节点（如 :id）
        if child := node.paramChild(); child != nil {
            // 提取参数值
            params.Add(child.paramName, seg)
            node = child
            continue
        }
        
        // 未找到匹配，返回 404
        return nil, nil
    }
    
    return node.route, params
}
```

## 完整输出格式

```yaml
CodeExplanation:
  explanation: string
  key_points: array
  design_patterns: array
  complexity:
    time: string
    space: string
  related_concepts: array
  prerequisites: array
```

## 写作风格指南

### 1. 先给结论，再给细节

```markdown
<!-- 好的写法 -->
`matchRoute` 使用前缀树实现 O(n) 时间复杂度的路由匹配。

具体实现中，前缀树的每个节点代表路径中的一个段...

<!-- 不好的写法 -->
前缀树是一种树形数据结构，每个节点代表...

在 `matchRoute` 函数中使用了前缀树来匹配路由。
```

### 2. 用比喻解释抽象概念

```markdown
前缀树就像图书馆的目录系统：
- 第一层按大类分（文学、科技、历史）
- 第二层按作者分
- 最后定位到具体的书籍

类似地，路由树按路径段逐级定位到处理器。
```

### 3. 保持技术准确性

- 时间/空间复杂度标注
- 边界条件说明
- 错误处理逻辑

## 使用示例

```yaml
# 在 WriterAgent 中使用
skills:
  - code-explanation

task:
  name: 解释代码
  steps:
    - action: code-explanation.explain
      input:
        code_snippet: "{{code_snippet}}"
        context: "{{title_context}}"
        target_audience: "中级开发者"
      output: code_explanation
```

## 依赖

- code.parse_ast
- code.get_snippet
- generation.llm_generate

## 最佳实践

1. 根据目标读者调整解释深度
2. 关键算法要说明复杂度
3. 设计模式识别要给出证据
4. 代码注释应简洁明了，解释"为什么"而非"做什么"
