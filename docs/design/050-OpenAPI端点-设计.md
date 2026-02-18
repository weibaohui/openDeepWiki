# OpenAPI 端点设计文档

## 设计编号
050

## 变更记录表

| 日期 | 版本 | 变更内容 | 变更人 |
|------|------|----------|--------|
| 2026-02-18 | 1.0 | 初始版本 | AI |

## 1. 设计概述

### 1.1 设计目标

提供标准化的 OpenAPI 规范文档，通过 `.well-known/openapi.yaml` 端点对外提供，使 AI 工具和其他开发者能够轻松理解和使用 openDeepWiki 的 API。

### 1.2 核心设计思路

1. **静态生成 + 动态路由**：预先生成完整的 OpenAPI YAML 文件，通过路由直接返回，避免运行时扫描和解析
2. **版本化管理**：OpenAPI 文档随 API 版本管理，确保文档与代码同步
3. **分层定义**：按功能模块组织 API 定义，便于维护和扩展
4. **AI 友好**：优化 API 路径、参数名称和响应格式，使 AI 更容易理解

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     HTTP Client                           │
│          (AI Tools / Browsers / API Clients)              │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
                  ┌───────────────┐
                  │ Gin Router    │
                  │               │
                  │ /.well-known/ │
                  │   openapi.yaml│
                  └───────┬───────┘
                          │
                          ▼
                  ┌───────────────┐
                  │OpenAPI Handler│
                  │               │
                  │ Read YAML File│
                  │ Return Content│
                  └───────┬───────┘
                          │
                          ▼
          ┌───────────────────────────────┐
          │   backend/.well-known/       │
          │   openapi.yaml               │
          │   (预生成的 OpenAPI 文档)    │
          └───────────────────────────────┘
```

### 2.2 目录结构

```
backend/
├── .well-known/              # OpenAPI 文档目录
│   └── openapi.yaml         # OpenAPI 规范文件
├── internal/
│   ├── handler/
│   │   └── openapi.go      # OpenAPI 处理器（新增）
│   └── router/
│       └── router.go       # 路由配置（修改）
└── pkg/
    └── openapi/            # OpenAPI 生成工具包（可选）
        └── generator.go   # OpenAPI 生成器
```

## 3. 详细设计

### 3.1 OpenAPI 处理器设计

#### 3.1.1 OpenAPIHandler 结构

```go
type OpenAPIHandler struct {
    // openAPIPath OpenAPI 文档文件路径
    openAPIPath string
}
```

#### 3.1.2 OpenAPIHandler 方法

```go
// NewOpenAPIHandler 创建 OpenAPI 处理器
// openAPIPath: OpenAPI 文档的文件路径
func NewOpenAPIHandler(openAPIPath string) *OpenAPIHandler

// ServeOpenAPI 提供 OpenAPI 规范文档
// 端点: /.well-known/openapi.yaml
// 方法: GET
// 响应: application/x-yaml
func (h *OpenAPIHandler) ServeOpenAPI(c *gin.Context)
```

### 3.2 路由配置设计

在 `internal/router/router.go` 中添加 OpenAPI 路由：

```go
// 在 Setup 函数中添加
openapiHandler := handler.NewOpenAPIHandler(".well-known/openapi.yaml")
r.GET("/.well-known/openapi.yaml", openapiHandler.ServeOpenAPI)
```

### 3.3 OpenAPI 文档结构设计

```yaml
openapi: 3.0.3
info:
  title: openDeepWiki API
  description: |
    openDeepWiki 是一个基于 AI 的代码仓库智能解读平台。

    ## AI 友好设计

    本 API 针对以下 AI 工具进行了优化：
    - Claude Code (Anthropic)
    - GitHub Copilot
    - 其他支持 OpenAPI 的 AI 工具

    ## 响应格式

    所有 API 响应遵循统一格式：

    ### 成功响应
    ```json
    {
      "code": 0,
      "message": "success",
      "data": { ... }
    }
    ```

    ### 错误响应
    ```json
    {
      "error": "error message"
    }
    ```
  version: 1.0.0
  contact:
    name: openDeepWiki
    url: https://github.com/weibaohui/opendeepwiki

servers:
  - url: http://localhost:8080
    description: 开发环境
  - url: https://opendeepwiki.example.com
    description: 生产环境

tags:
  - name: repositories
    description: 仓库管理
  - name: tasks
    description: 任务管理
  - name: documents
    description: 文档管理
  - name: api-keys
    description: API Key 管理
  - name: sync
    description: 数据同步
  - name: user-requests
    description: 用户需求管理

paths:
  # 仓库管理 API
  /api/repositories:
    post:
      tags:
        - repositories
      summary: 创建仓库
      description: 添加一个新的代码仓库到系统
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateRepositoryRequest'
            example:
              url: https://github.com/user/repo.git
      responses:
        '201':
          description: 创建成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Repository'
        '400':
          description: 请求参数错误
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '409':
          description: 仓库已存在
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
    get:
      tags:
        - repositories
      summary: 获取仓库列表
      description: 获取系统中所有仓库的列表
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Repository'

  # ... 其他 API 定义

components:
  schemas:
    # 通用响应
    SuccessResponse:
      type: object
      properties:
        code:
          type: integer
          description: 响应码，0 表示成功
          example: 0
        message:
          type: string
          description: 响应消息
          example: success
        data:
          type: object
          description: 响应数据
      required:
        - code
        - message

    ErrorResponse:
      type: object
      properties:
        error:
          type: string
          description: 错误信息
      required:
        - error

    # 仓库相关
    Repository:
      type: object
      properties:
        id:
          type: integer
          description: 仓库 ID
        name:
          type: string
          description: 仓库名称
        url:
          type: string
          description: 仓库 URL
        status:
          type: string
          description: 仓库状态
        branch:
          type: string
          description: 当前分支
        size:
          type: integer
          description: 仓库大小（字节）
        created_at:
          type: string
          format: date-time
          description: 创建时间
        updated_at:
          type: string
          format: date-time
          description: 更新时间
      required:
        - id
        - name
        - url
        - status

    CreateRepositoryRequest:
      type: object
      properties:
        url:
          type: string
          description: 仓库 URL（支持 https:// 和 git@ 格式）
          example: https://github.com/user/repo.git
      required:
        - url

    # 任务相关
    Task:
      type: object
      properties:
        id:
          type: integer
          description: 任务 ID
        repository_id:
          type: integer
          description: 关联的仓库 ID
        title:
          type: string
          description: 任务标题
        status:
          type: string
          description: 任务状态
          enum:
            - pending
            - queued
            - running
            - success
            - failed
            - canceled
        writer_name:
          type: string
          description: 写入器名称
        sort_order:
          type: integer
          description: 排序顺序
        error:
          type: string
          description: 错误信息
        created_at:
          type: string
          format: date-time
          description: 创建时间
        updated_at:
          type: string
          format: date-time
          description: 更新时间
      required:
        - id
        - repository_id
        - title
        - status

    # 文档相关
    Document:
      type: object
      properties:
        id:
          type: integer
          description: 文档 ID
        repository_id:
          type: integer
          description: 关联的仓库 ID
        task_id:
          type: integer
          description: 关联的任务 ID
        title:
          type: string
          description: 文档标题
        content:
          type: string
          description: 文档内容（Markdown 格式）
        version:
          type: integer
          description: 文档版本号
        created_at:
          type: string
          format: date-time
          description: 创建时间
        updated_at:
          type: string
          format: date-time
          description: 更新时间
      required:
        - id
        - repository_id
        - title
        - content

    # API Key 相关
    APIKey:
      type: object
      properties:
        id:
          type: integer
          description: API Key ID
        name:
          type: string
          description: API Key 名称
        provider:
          type: string
          description: 提供商
        base_url:
          type: string
          description: API 地址
        api_key:
          type: string
          description: API Key（脱敏）
        model:
          type: string
          description: 模型名称
        priority:
          type: integer
          description: 优先级
        status:
          type: string
          description: 状态
          enum:
            - enabled
            - disabled
        request_count:
          type: integer
          description: 请求次数
        error_count:
          type: integer
          description: 错误次数
        created_at:
          type: string
          format: date-time
          description: 创建时间
        updated_at:
          type: string
          format: date-time
      required:
        - id
        - name
        - provider
        - base_url
        - model

    # 用户需求相关
    UserRequest:
      type: object
      properties:
        id:
          type: integer
          description: 需求 ID
        repository_id:
          type: integer
          description: 关联的仓库 ID
        content:
          type: string
          description: 需求内容
        status:
          type: string
          description: 状态
          enum:
            - pending
            - processing
            - completed
        created_at:
          type: string
          format: date-time
          description: 创建时间
        updated_at:
          type: string
          format: date-time
          description: 更新时间
      required:
        - id
        - repository_id
        - content
        - status
```

## 4. API 优化设计

### 4.1 路径和参数语义优化

为提高 AI 友好性，对以下 API 进行说明：

| 原路径 | 语义说明 |
|--------|----------|
| `/api/repositories/:id/run-all` | 执行仓库的所有分析任务 |
| `/api/repositories/:id/clone` | 重新克隆仓库代码 |
| `/api/repositories/:id/purge-local` | 清空仓库的本地缓存 |
| `/api/repositories/:id/set-ready` | 将仓库状态设置为就绪，允许执行任务 |

### 4.2 响应格式统一

确保所有 API 遵循统一的响应格式：

```go
// 成功响应
type SuccessResponse struct {
    Code    int         `json:"code"`    // 0 表示成功
    Message string      `json:"message"` // 响应消息
    Data    interface{} `json:"data"`    // 响应数据
}

// 错误响应（简化版）
type ErrorResponse struct {
    Error string `json:"error"` // 错误信息
}
```

## 5. 实现步骤

1. **创建 OpenAPI 处理器**
   - 创建 `internal/handler/openapi.go`
   - 实现 OpenAPIHandler 结构和方法

2. **添加路由配置**
   - 修改 `internal/router/router.go`
   - 添加 `/.well-known/openapi.yaml` 路由

3. **生成 OpenAPI 文档**
   - 创建 `backend/.well-known/openapi.yaml`
   - 填写完整的 OpenAPI 规范内容

4. **测试验证**
   - 测试 OpenAPI 端点可访问性
   - 验证 OpenAPI 文档格式正确性
   - 使用 Swagger UI 或 Redoc 验证文档可渲染性

## 6. 后续扩展

1. **动态生成**：考虑使用注解或代码扫描动态生成 OpenAPI 文档
2. **多版本支持**：支持多个 API 版本的 OpenAPI 文档
3. **文档托管**：集成 Swagger UI，提供交互式文档页面
4. **SDK 生成**：基于 OpenAPI 文档自动生成客户端 SDK

## 7. 安全考虑

1. OpenAPI 文档不包含敏感信息（如真实 API Key）
2. API Key 返回时自动脱敏
3. 不暴露内部实现细节
