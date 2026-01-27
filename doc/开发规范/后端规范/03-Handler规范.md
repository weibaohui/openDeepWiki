# Handler 规范

本文档定义 Gin Handler 的编写规范。

## Handler 职责

Handler **只能做 5 件事**：

1. 参数绑定（Bind）
2. 参数校验（Validate）
3. 调用 Service
4. 错误转换
5. 返回统一 Response

## 标准模板

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "project/internal/service"
    "project/internal/pkg/response"
)

type UserHandler struct {
    userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
    // 1. 参数绑定
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err)
        return
    }

    // 2. 参数校验（如果 binding tag 不够用）
    if err := req.Validate(); err != nil {
        response.BadRequest(c, err)
        return
    }

    // 3. 调用 Service
    result, err := h.userService.Create(c.Request.Context(), req)
    
    // 4. 错误转换
    if err != nil {
        response.FromError(c, err)
        return
    }

    // 5. 返回统一 Response
    response.OK(c, result)
}
```

## 参数绑定

### JSON Body

```go
var req CreateUserRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.BadRequest(c, err)
    return
}
```

### URL 参数

```go
id := c.Param("id")
if id == "" {
    response.BadRequest(c, errors.New("id is required"))
    return
}
```

### Query 参数

```go
var query ListUserQuery
if err := c.ShouldBindQuery(&query); err != nil {
    response.BadRequest(c, err)
    return
}
```

## Request 结构定义

```go
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Name     string `json:"name" binding:"required,min=2,max=50"`
    Password string `json:"password" binding:"required,min=8"`
}

// Validate 自定义校验（binding tag 不够用时）
func (r *CreateUserRequest) Validate() error {
    // 复杂校验逻辑
    return nil
}
```

## Context 传递

始终使用 `c.Request.Context()` 传递 context：

```go
result, err := h.userService.Create(c.Request.Context(), req)
```

## 路由注册

```go
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
    users := r.Group("/users")
    {
        users.POST("", h.Create)
        users.GET("", h.List)
        users.GET("/:id", h.Get)
        users.PUT("/:id", h.Update)
        users.DELETE("/:id", h.Delete)
    }
}
```

## 禁止事项

```go
// ❌ 在 handler 中写业务判断
func (h *UserHandler) Create(c *gin.Context) {
    if user.Age < 18 {  // 业务逻辑应在 Service
        response.BadRequest(c, errors.New("age must be >= 18"))
        return
    }
}

// ❌ 在 handler 中操作数据库
func (h *UserHandler) Create(c *gin.Context) {
    h.db.Create(&user)  // 禁止！
}

// ❌ 在 handler 中开启事务
func (h *UserHandler) Create(c *gin.Context) {
    tx := h.db.Begin()  // 禁止！
}

// ❌ 直接返回 Model
func (h *UserHandler) Get(c *gin.Context) {
    response.OK(c, userModel)  // 应返回 DTO
}
```

## 相关文档

- [Service 规范](./04-Service规范.md)
- [Response 规范](./12-Response规范.md)
- [DTO 规范](./11-DTO规范.md)
