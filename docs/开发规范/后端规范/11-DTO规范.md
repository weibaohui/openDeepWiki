# DTO 规范

本文档定义 Request / DTO / Response 的区分和使用规范。

## 三种结构体必须区分

| 类型    | 用途         | 所在层     | Tag           |
| ------- | ------------ | ---------- | ------------- |
| Request | HTTP 入参    | Handler    | json, binding |
| DTO     | Service 返回 | Service    | json          |
| Model   | DB / GORM    | Repository | gorm          |

**禁止混用**。

## Request 定义

```go
package handler

// CreateUserRequest HTTP 创建用户请求
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Name     string `json:"name" binding:"required,min=2,max=50"`
    Password string `json:"password" binding:"required,min=8"`
}

// UpdateUserRequest HTTP 更新用户请求
type UpdateUserRequest struct {
    Name   string `json:"name" binding:"omitempty,min=2,max=50"`
    Status string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// ListUserQuery HTTP 列表查询参数
type ListUserQuery struct {
    Page     int    `form:"page" binding:"omitempty,min=1"`
    PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
    Status   string `form:"status" binding:"omitempty,oneof=active inactive"`
}
```

## DTO 定义

```go
package service

import "time"

// UserDTO Service 返回的用户数据
type UserDTO struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

// ListUserResult 列表查询结果
type ListUserResult struct {
    Items []*UserDTO `json:"items"`
    Total int64      `json:"total"`
    Page  int        `json:"page"`
    Size  int        `json:"size"`
}
```

## 数据流向

```
Request (Handler) 
    ↓ 转换
Domain (Service)
    ↓ 转换
Model (Repository)
    ↓ DB 操作
Model (Repository)
    ↓ 转换
Domain (Service)
    ↓ 转换
DTO (Service)
    ↓
Response (Handler)
```

## 转换函数

### Request → Domain（在 Service 中）

```go
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*UserDTO, error) {
    // Request → Domain
    user := &domain.User{
        ID:    generateID(),
        Email: req.Email,
        Name:  req.Name,
    }
    // ...
}
```

### Domain → DTO（在 Service 中）

```go
func toUserDTO(user *domain.User) *UserDTO {
    return &UserDTO{
        ID:        user.ID,
        Email:     user.Email,
        Name:      user.Name,
        Status:    string(user.Status),
        CreatedAt: user.CreatedAt,
    }
}

func toUserDTOList(users []*domain.User) []*UserDTO {
    result := make([]*UserDTO, len(users))
    for i, user := range users {
        result[i] = toUserDTO(user)
    }
    return result
}
```

### Domain ↔ Model（在 Repository 中）

```go
// Domain → Model
func toModel(user *domain.User) *model.UserModel {
    return &model.UserModel{
        ID:        user.ID,
        Email:     user.Email,
        Name:      user.Name,
        Status:    string(user.Status),
    }
}

// Model → Domain
func toDomain(m *model.UserModel) *domain.User {
    return &domain.User{
        ID:        m.ID,
        Email:     m.Email,
        Name:      m.Name,
        Status:    domain.UserStatus(m.Status),
    }
}
```

## 禁止事项

```go
// ❌ Request 结构体包含 gorm tag
type CreateUserRequest struct {
    Email string `json:"email" gorm:"uniqueIndex"`  // 禁止！
}

// ❌ Model 直接返回给 Handler
func (s *userService) GetByID(ctx context.Context, id string) (*model.UserModel, error) {
    // 禁止！应返回 DTO
}

// ❌ Service 方法参数使用 Model
func (s *userService) Create(ctx context.Context, m *model.UserModel) error {
    // 禁止！应使用 Request 或 Domain
}

// ❌ 一个结构体多用途
type User struct {
    ID    string `json:"id" gorm:"primaryKey" binding:"required"`  // 禁止混用！
}
```

## 命名规范

| 类型     | 命名规范         | 示例              |
| -------- | ---------------- | ----------------- |
| 创建请求 | CreateXxxRequest | CreateUserRequest |
| 更新请求 | UpdateXxxRequest | UpdateUserRequest |
| 查询参数 | ListXxxQuery     | ListUserQuery     |
| DTO      | XxxDTO           | UserDTO           |
| 列表结果 | ListXxxResult    | ListUserResult    |
| Model    | XxxModel         | UserModel         |

## 相关文档

- [Handler 规范](./03-Handler规范.md)
- [Service 规范](./04-Service规范.md)
- [Response 规范](./12-Response规范.md)
