---
name: example-generation
description: 生成使用示例。生成最小可运行示例、使用场景示例、边界情况示例。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: writing
  priority: P1
---

# Example Generation Skill

生成代码使用示例，帮助读者理解如何使用代码。

## 使用场景

- WriterAgent 为代码添加示例
- 生成 API 使用示例
- 展示边界情况处理

## 功能能力

### 1. generate_minimal_example

生成最小可运行示例。

**输入：**
```yaml
code_context:
  package: "router"
  function: "matchRoute"
  signature: "func (r *Router) matchRoute(path string) (*Route, Params)"
```

**输出：**
```yaml
minimal_example:
  description: "基本路由匹配示例"
  code: |
    package main
    
    import (
        "fmt"
        "myapp/router"
    )
    
    func main() {
        // 创建路由器
        r := router.New()
        
        // 注册路由
        r.GET("/user/:id", getUserHandler)
        
        // 匹配请求
        route, params := r.Match("/user/123")
        
        fmt.Println("Route:", route.Path)
        fmt.Println("User ID:", params.Get("id"))
        // 输出:
        // Route: /user/:id
        // User ID: 123
    }
  
  key_points:
    - "使用 router.New() 创建路由器实例"
    - "使用 r.GET() 注册路由和处理函数"
    - "使用 r.Match() 匹配请求路径"
  
  expected_output: |
    Route: /user/:id
    User ID: 123
```

### 2. generate_usage_scenario

生成使用场景示例。

**场景类型：**
- 典型用例（Happy Path）
- 错误处理
- 批量操作
- 并发场景

**输出：**
```yaml
scenario_example:
  scenario: "REST API 路由配置"
  description: |
    在一个博客系统中，配置文章相关的 REST API 路由。
  
  code: |
    func setupRoutes(r *router.Router) {
        // 文章列表（分页）
        r.GET("/api/posts", listPostsHandler)
        
        // 获取单篇文章
        r.GET("/api/posts/:id", getPostHandler)
        
        // 创建文章（需要认证）
        r.POST("/api/posts", authMiddleware, createPostHandler)
        
        // 更新文章
        r.PUT("/api/posts/:id", authMiddleware, updatePostHandler)
        
        // 删除文章
        r.DELETE("/api/posts/:id", authMiddleware, deletePostHandler)
        
        // 嵌套资源：文章评论
        r.GET("/api/posts/:id/comments", listCommentsHandler)
        r.POST("/api/posts/:id/comments", authMiddleware, addCommentHandler)
    }
  
  explanation: |
    这个示例展示了如何为一个资源设计完整的 REST API：
    
    1. **集合操作**: `GET /api/posts` 获取列表
    2. **单资源操作**: `GET/PUT/DELETE /api/posts/:id` 对单篇文章操作
    3. **中间件使用**: 在创建/更新/删除路由上添加认证中间件
    4. **嵌套资源**: `/api/posts/:id/comments` 表示文章下的评论
```

### 3. generate_edge_case_example

生成边界情况示例。

**边界情况类型：**
- 空输入
- 超长输入
- 特殊字符
- 并发访问
- 资源耗尽

**输出：**
```yaml
edge_case_examples:
  - case: "空路径"
    description: "请求根路径 '/'"
    code: |
      route, params := r.Match("/")
      // route 为 nil（如果没有注册根路由）
      // 或返回注册的处理函数
    
  - case: "特殊字符"
    description: "路径中包含 URL 编码字符"
    code: |
      // 请求: /user/john%40example.com
      route, params := r.Match("/user/john@example.com")
      // params.Get("id") = "john@example.com"
    
  - case: "路径冲突"
    description: "静态路由和动态路由冲突"
    code: |
      r.GET("/user/me", getCurrentUserHandler)
      r.GET("/user/:id", getUserHandler)
      
      // 匹配 /user/me
      // 优先匹配静态路由，返回 getCurrentUserHandler
      route, _ := r.Match("/user/me")
    
    explanation: |
      当静态路由 `/user/me` 和动态路由 `/user/:id` 同时存在时，
      路由器会优先匹配静态路由，确保特殊路径（如获取当前用户）
      不会被动态路由覆盖。
```

## 完整输出格式

```yaml
ExampleSet:
  minimal:
    description: string
    code: string
    key_points: array
    expected_output: string
    
  scenarios:
    - scenario: string
      description: string
      code: string
      explanation: string
      
  edge_cases:
    - case: string
      description: string
      code: string
      explanation: string
```

## 示例代码规范

### 1. 完整性

- 可以独立运行
- 包含必要的 import
- 有预期的输出

### 2. 简洁性

- 去掉无关代码
- 使用有意义的变量名
- 添加关键注释

### 3. 可读性

- 适当的空行
- 一致的缩进
- 输出结果注释

## 使用示例

```yaml
# 在 WriterAgent 中使用
skills:
  - example-generation

task:
  name: 生成代码示例
  steps:
    - action: example-generation.generate
      input:
        code_context: "{{function_info}}"
        example_types: ["minimal", "scenario", "edge_cases"]
      output: examples
```

## 依赖

- code.parse_ast
- generation.llm_generate

## 最佳实践

1. 示例代码应该能直接运行或稍作修改就能运行
2. 输出结果用注释标明
3. 复杂示例要分步骤解释
4. 边界情况示例帮助读者避免常见错误
