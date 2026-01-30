---
name: diagram-description
description: 生成图表描述文字（Mermaid等）。生成流程图、时序图、类图、架构图描述。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: writing
  priority: P1
---

# Diagram Description Skill

将代码逻辑转化为 Mermaid 图表描述。

## 使用场景

- WriterAgent 为代码生成可视化图表
- 展示系统架构
- 说明执行流程

## 功能能力

### 1. generate_flowchart

生成流程图描述。

**输入：**
```yaml
code_snippet: |
  func (r *Router) matchRoute(path string) (*Route, Params) {
      segments := splitPath(path)
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
flowchart:
  description: |
    路由匹配流程：首先分割路径，然后遍历前缀树，
    优先匹配静态节点，其次匹配动态参数节点。
  
  mermaid_code: |
    flowchart TD
        A[开始] --> B[分割路径]
        B --> C{还有路径段?}
        C -->|是| D[获取当前段]
        D --> E{存在静态子节点?}
        E -->|是| F[移动到静态节点]
        E -->|否| G{存在动态参数节点?}
        G -->|是| H[提取参数值]
        H --> I[移动到参数节点]
        G -->|否| J[返回 404]
        F --> C
        I --> C
        C -->|否| K[返回匹配的路由]
        K --> L[结束]
        J --> L
  
  key_steps:
    - "路径分割: 将 URL 按 '/' 分割"
    - "静态匹配: 优先查找同名子节点"
    - "动态匹配: 提取参数值后匹配"
    - "返回结果: 成功返回路由，失败返回 nil"
```

### 2. generate_sequence_diagram

生成时序图描述。

**输入：**
```yaml
scenario: "HTTP 请求处理流程"
participants:
  - Client
  - Server
  - Router
  - Handler
  - Database
```

**输出：**
```yaml
sequence_diagram:
  description: "HTTP 请求的处理时序"
  
  mermaid_code: |
    sequenceDiagram
        participant C as Client
        participant S as Server
        participant R as Router
        participant H as Handler
        participant D as Database
        
        C->>S: HTTP Request
        S->>R: matchRoute(path)
        R-->>S: Route, Params
        S->>H: Handler(Context)
        H->>D: Query
        D-->>H: Data
        H-->>S: Response
        S-->>C: HTTP Response
  
  explanation: |
    1. **Client → Server**: 发送 HTTP 请求
    2. **Server → Router**: 请求路由匹配
    3. **Router → Server**: 返回匹配的路由和参数
    4. **Server → Handler**: 调用对应的处理器
    5. **Handler → Database**: 查询数据（如有需要）
    6. **Handler → Server**: 返回处理结果
    7. **Server → Client**: 发送 HTTP 响应
```

### 3. generate_class_diagram

生成类图描述。

**输入：**
```yaml
classes:
  - name: "Router"
    fields:
      - "root *node"
      - "routes map[string]*Route"
    methods:
      - "GET(path string, handler Handler)"
      - "POST(path string, handler Handler)"
      - "matchRoute(path string) (*Route, Params)"
      
  - name: "Route"
    fields:
      - "path string"
      - "handler Handler"
      - "middleware []Middleware"
      
  - name: "node"
    fields:
      - "path string"
      - "children map[string]*node"
      - "paramChild *node"
      - "route *Route"
```

**输出：**
```yaml
class_diagram:
  description: "路由系统的类结构"
  
  mermaid_code: |
    classDiagram
        class Router {
            -root *node
            -routes map[string]*Route
            +GET(path string, handler Handler)
            +POST(path string, handler Handler)
            +matchRoute(path string) (*Route, Params)
        }
        
        class Route {
            +path string
            +handler Handler
            +middleware []Middleware
        }
        
        class node {
            -path string
            -children map[string]*node
            -paramChild *node
            -route *Route
        }
        
        class Params {
            +Get(key string) string
            +Add(key, value string)
        }
        
        Router "1" --> "*" Route : manages
        Router "1" --> "1" node : root
        node "1" --> "*" node : children
        Route "1" --> "1" Params : uses
  
  relationships:
    - "Router 管理多个 Route"
    - "Router 拥有根 node"
    - "node 通过 children 形成树结构"
```

### 4. generate_architecture_diagram

生成架构图描述。

**输出：**
```yaml
architecture_diagram:
  description: "系统整体架构"
  
  mermaid_code: |
    graph TB
        Client[客户端] -->|HTTP| API[API Gateway]
        
        subgraph "Web 服务"
            API --> Router[路由层]
            Router --> Middleware[中间件链]
            Middleware --> Handler[处理器层]
            Handler --> Service[服务层]
            Service --> Repository[数据访问层]
        end
        
        Repository -->|SQL| Database[(数据库)]
        Repository -->|Redis| Cache[(缓存)]
        
        style API fill:#f9f,stroke:#333
        style Router fill:#bbf,stroke:#333
        style Service fill:#bfb,stroke:#333
  
  layers:
    - name: "入口层"
      components: ["API Gateway", "Router"]
      
    - name: "处理层"
      components: ["Middleware", "Handler"]
      
    - name: "业务层"
      components: ["Service"]
      
    - name: "数据层"
      components: ["Repository", "Database", "Cache"]
```

## 图表类型选择指南

| 场景 | 推荐图表类型 |
|------|-------------|
| 算法流程 | flowchart |
| 请求处理过程 | sequenceDiagram |
| 类结构关系 | classDiagram |
| 系统组件关系 | graph (架构图) |
| 状态转换 | stateDiagram |
| 时间线 | timeline |

## 使用示例

```yaml
# 在 WriterAgent 中使用
skills:
  - diagram-description

task:
  name: 生成图表
  steps:
    - action: diagram-description.generate_flowchart
      input:
        code_snippet: "{{code_snippet}}"
        description: "路由匹配流程"
      output: flowchart
```

## 依赖

- code.parse_ast
- code.get_call_graph

## 最佳实践

1. 图表不要太复杂，超过 10 个节点考虑拆分
2. 使用有意义的节点名称
3. 复杂图表添加说明文字
4. 保持图表风格一致
