# Response 规范

本文档定义统一响应结构规范。

## 统一响应格式

所有接口必须返回此结构：

```json
{
  "code": "OK",
  "message": "success",
  "data": {}
}
```

## Response 结构定义

```go
package response

// Response 统一响应结构
type Response struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 状态码常量
const (
    CodeOK           = "OK"
    CodeBadRequest   = "BAD_REQUEST"
    CodeUnauthorized = "UNAUTHORIZED"
    CodeForbidden    = "FORBIDDEN"
    CodeNotFound     = "NOT_FOUND"
    CodeConflict     = "CONFLICT"
    CodeInternal     = "INTERNAL_ERROR"
)
```

## 响应函数

```go
package response

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    CodeOK,
        Message: "success",
        Data:    data,
    })
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
    c.JSON(http.StatusCreated, Response{
        Code:    CodeOK,
        Message: "created",
        Data:    data,
    })
}

// NoContent 无内容响应
func NoContent(c *gin.Context) {
    c.Status(http.StatusNoContent)
}

// BadRequest 请求错误响应
func BadRequest(c *gin.Context, err error) {
    c.JSON(http.StatusBadRequest, Response{
        Code:    CodeBadRequest,
        Message: err.Error(),
    })
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, err error) {
    c.JSON(http.StatusUnauthorized, Response{
        Code:    CodeUnauthorized,
        Message: err.Error(),
    })
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, err error) {
    c.JSON(http.StatusForbidden, Response{
        Code:    CodeForbidden,
        Message: err.Error(),
    })
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, err error) {
    c.JSON(http.StatusNotFound, Response{
        Code:    CodeNotFound,
        Message: err.Error(),
    })
}

// Conflict 冲突响应
func Conflict(c *gin.Context, err error) {
    c.JSON(http.StatusConflict, Response{
        Code:    CodeConflict,
        Message: err.Error(),
    })
}

// InternalError 内部错误响应
func InternalError(c *gin.Context, err error) {
    c.JSON(http.StatusInternalServerError, Response{
        Code:    CodeInternal,
        Message: "internal server error",  // 不暴露内部错误详情
    })
}
```

## 错误转换

```go
package response

import (
    "errors"

    "project/internal/service"
)

// FromError 根据错误类型返回对应响应
func FromError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, service.ErrUserNotFound):
        NotFound(c, err)
    case errors.Is(err, service.ErrUserAlreadyExists):
        Conflict(c, err)
    case errors.Is(err, service.ErrInvalidPassword):
        BadRequest(c, err)
    case errors.Is(err, service.ErrPermissionDenied):
        Forbidden(c, err)
    default:
        InternalError(c, err)
    }
}
```

## 分页响应

```go
// ListResponse 分页列表响应
type ListResponse struct {
    Items interface{} `json:"items"`
    Total int64       `json:"total"`
    Page  int         `json:"page"`
    Size  int         `json:"size"`
}

// OKList 列表成功响应
func OKList(c *gin.Context, items interface{}, total int64, page, size int) {
    c.JSON(http.StatusOK, Response{
        Code:    CodeOK,
        Message: "success",
        Data: ListResponse{
            Items: items,
            Total: total,
            Page:  page,
            Size:  size,
        },
    })
}
```

## 在 Handler 中使用

```go
func (h *UserHandler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err)
        return
    }

    result, err := h.userService.Create(c.Request.Context(), req)
    if err != nil {
        response.FromError(c, err)
        return
    }

    response.Created(c, result)
}

func (h *UserHandler) List(c *gin.Context) {
    var query ListUserQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        response.BadRequest(c, err)
        return
    }

    result, err := h.userService.List(c.Request.Context(), query)
    if err != nil {
        response.FromError(c, err)
        return
    }

    response.OKList(c, result.Items, result.Total, query.Page, query.PageSize)
}
```

## 禁止事项

```go
// ❌ 不统一的响应格式
c.JSON(200, gin.H{"user": user})  // 禁止！

// ❌ 直接返回错误详情
c.JSON(500, gin.H{"error": err.Error()})  // 禁止暴露内部错误！

// ❌ 不同接口返回不同结构
c.JSON(200, user)  // 禁止！
c.JSON(200, Response{Data: user})  // 另一个接口
```

## 相关文档

- [Handler 规范](./03-Handler规范.md)
- [错误处理规范](./05-错误处理规范.md)
