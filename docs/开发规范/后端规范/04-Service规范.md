# Service 规范

本文档定义 Service 层的编写规范。

## Service 职责

Service 是**业务逻辑的唯一入口**：

- 业务规则校验
- 多 Repository 协调
- 事务控制
- Domain 与 DTO 转换

## 接口定义

```go
package service

import (
    "context"
)

type UserService interface {
    Create(ctx context.Context, req CreateUserRequest) (*UserDTO, error)
    GetByID(ctx context.Context, id string) (*UserDTO, error)
    List(ctx context.Context, query ListUserQuery) (*ListUserResult, error)
    Update(ctx context.Context, id string, req UpdateUserRequest) (*UserDTO, error)
    Delete(ctx context.Context, id string) error
}
```

## 实现模板

```go
package service

import (
    "context"
    "fmt"

    "project/internal/domain"
    "project/internal/repository"
)

type userService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
    return &userService{userRepo: userRepo}
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*UserDTO, error) {
    // 1. 业务规则校验
    existing, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil && !errors.Is(err, domain.ErrRecordNotFound) {
        return nil, fmt.Errorf("failed to check email: %w", err)
    }
    if existing != nil {
        return nil, ErrUserAlreadyExists
    }

    // 2. 构建领域对象
    user := &domain.User{
        ID:    generateID(),
        Email: req.Email,
        Name:  req.Name,
    }

    // 3. 调用 Repository
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    // 4. 转换为 DTO 返回
    return toUserDTO(user), nil
}
```

## 方法签名规则

### 必须传入 Context

```go
// 正确
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*UserDTO, error)

// 错误
func (s *userService) Create(req CreateUserRequest) (*UserDTO, error)
```

### 返回值统一为 (Result, error)

```go
// 正确
func (s *userService) GetByID(ctx context.Context, id string) (*UserDTO, error)
func (s *userService) List(ctx context.Context, query ListUserQuery) (*ListUserResult, error)
func (s *userService) Delete(ctx context.Context, id string) error

// 错误：不返回 error
func (s *userService) GetByID(ctx context.Context, id string) *UserDTO
```

## DTO 转换

```go
// domain.User → UserDTO
func toUserDTO(user *domain.User) *UserDTO {
    return &UserDTO{
        ID:        user.ID,
        Email:     user.Email,
        Name:      user.Name,
        CreatedAt: user.CreatedAt,
    }
}

// []domain.User → []*UserDTO
func toUserDTOList(users []*domain.User) []*UserDTO {
    result := make([]*UserDTO, len(users))
    for i, user := range users {
        result[i] = toUserDTO(user)
    }
    return result
}
```

## 业务错误定义

```go
package service

import "errors"

var (
    ErrUserAlreadyExists = errors.New("user already exists")
    ErrUserNotFound      = errors.New("user not found")
    ErrInvalidPassword   = errors.New("invalid password")
)
```

## 禁止事项

```go
// ❌ 依赖 Gin
func (s *userService) Create(c *gin.Context, req CreateUserRequest) (*UserDTO, error)

// ❌ 直接使用 GORM
func (s *userService) Create(ctx context.Context, req CreateUserRequest) (*UserDTO, error) {
    s.db.Create(&model)  // 禁止！
}

// ❌ 返回 GORM Model
func (s *userService) GetByID(ctx context.Context, id string) (*model.UserModel, error)

// ❌ 不传 Context
func (s *userService) Create(req CreateUserRequest) (*UserDTO, error)
```

## 相关文档

- [Handler 规范](./03-Handler规范.md)
- [Repository 规范](./07-Repository规范.md)
- [事务规范](./10-事务规范.md)
