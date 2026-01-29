# Repository 规范

本文档定义数据访问层的编写规范。

## Repository 职责

Repository 是**唯一允许使用 GORM 的地方**：

- CRUD 操作
- 数据查询
- Model 与 Domain 转换

## 接口定义

```go
package repository

import (
    "context"
    "project/internal/domain"
)

type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id string) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    List(ctx context.Context, opts ListOptions) ([]*domain.User, int64, error)
    Update(ctx context.Context, user *domain.User) error
    Delete(ctx context.Context, id string) error
}

// ListOptions 列表查询选项
type ListOptions struct {
    Offset int
    Limit  int
    Status string
}
```

## 实现模板

```go
package repository

import (
    "context"
    "errors"
    "fmt"

    "gorm.io/gorm"
    "project/internal/domain"
    "project/internal/model"
)

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    m := toModel(user)
    if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return toDomain(&m), nil
}

func (r *userRepository) List(ctx context.Context, opts ListOptions) ([]*domain.User, int64, error) {
    var models []model.UserModel
    var total int64

    query := r.db.WithContext(ctx).Model(&model.UserModel{})
    
    if opts.Status != "" {
        query = query.Where("status = ?", opts.Status)
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count users: %w", err)
    }

    if err := query.Offset(opts.Offset).Limit(opts.Limit).Find(&models).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to list users: %w", err)
    }

    return toDomainList(models), total, nil
}
```

## Model ↔ Domain 转换

```go
// Domain → Model
func toModel(user *domain.User) *model.UserModel {
    return &model.UserModel{
        ID:        user.ID,
        Email:     user.Email,
        Name:      user.Name,
        Status:    string(user.Status),
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
    }
}

// Model → Domain
func toDomain(m *model.UserModel) *domain.User {
    return &domain.User{
        ID:        m.ID,
        Email:     m.Email,
        Name:      m.Name,
        Status:    domain.UserStatus(m.Status),
        CreatedAt: m.CreatedAt,
        UpdatedAt: m.UpdatedAt,
    }
}

// []Model → []Domain
func toDomainList(models []model.UserModel) []*domain.User {
    result := make([]*domain.User, len(models))
    for i := range models {
        result[i] = toDomain(&models[i])
    }
    return result
}
```

## 错误定义

```go
package repository

import "errors"

var (
    ErrNotFound     = errors.New("record not found")
    ErrDuplicateKey = errors.New("duplicate key")
)
```

## 显式处理 ErrRecordNotFound

```go
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
    
    // 必须显式处理
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return toDomain(&m), nil
}
```

## 禁止事项

```go
// ❌ 返回 Model
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.UserModel, error)

// ❌ 不使用 WithContext
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    r.db.Where("id = ?", id).First(&m)  // 禁止！
}

// ❌ 开启事务
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    tx := r.db.Begin()  // 禁止！
}

// ❌ 链式自由拼查询
func (r *userRepository) Search(ctx context.Context, q string) ([]*domain.User, error) {
    r.db.Where("name LIKE ?", "%"+q+"%").Or("email LIKE ?", "%"+q+"%")  // 避免
}
```

## 相关文档

- [Model 规范](./08-Model规范.md)
- [GORM 使用规范](./09-GORM使用规范.md)
- [事务规范](./10-事务规范.md)
