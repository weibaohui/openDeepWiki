# Domain 规范

本文档定义领域模型的编写规范。

## Domain 职责

Domain 只包含**不变的业务事实**：

- 领域实体（Entity）
- 值对象（Value Object）
- 业务规则

## 领域实体定义

```go
package domain

import "time"

// User 用户领域实体
type User struct {
    ID        string
    Email     string
    Name      string
    Status    UserStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}

// UserStatus 用户状态（值对象）
type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusInactive UserStatus = "inactive"
    UserStatusBanned   UserStatus = "banned"
)
```

## 业务规则

业务规则可以作为 Domain 的方法：

```go
// IsActive 判断用户是否激活
func (u *User) IsActive() bool {
    return u.Status == UserStatusActive
}

// CanLogin 判断用户是否可以登录
func (u *User) CanLogin() bool {
    return u.Status == UserStatusActive
}

// Validate 校验用户数据
func (u *User) Validate() error {
    if u.Email == "" {
        return errors.New("email is required")
    }
    if u.Name == "" {
        return errors.New("name is required")
    }
    return nil
}
```

## 值对象

```go
package domain

// Email 邮箱值对象
type Email string

func NewEmail(s string) (Email, error) {
    // 校验邮箱格式
    if !isValidEmail(s) {
        return "", errors.New("invalid email format")
    }
    return Email(s), nil
}

func (e Email) String() string {
    return string(e)
}

// Money 金额值对象
type Money struct {
    Amount   int64
    Currency string
}

func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, errors.New("currency mismatch")
    }
    return Money{
        Amount:   m.Amount + other.Amount,
        Currency: m.Currency,
    }, nil
}
```

## 聚合根

```go
package domain

// Repository 仓库聚合根
type Repository struct {
    ID          string
    Name        string
    URL         string
    Status      RepositoryStatus
    Tasks       []Task      // 聚合内的实体
    Documents   []Document  // 聚合内的实体
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// AddTask 添加任务（聚合根方法）
func (r *Repository) AddTask(task Task) error {
    // 业务规则校验
    if r.Status == RepositoryStatusError {
        return errors.New("cannot add task to error repository")
    }
    r.Tasks = append(r.Tasks, task)
    return nil
}
```

## 禁止事项

```go
// ❌ 依赖 GORM
type User struct {
    ID    string `gorm:"primaryKey"`  // 禁止！
    Email string `gorm:"uniqueIndex"` // 禁止！
}

// ❌ 依赖 Gin / JSON
type User struct {
    ID    string `json:"id"`    // 禁止！
    Email string `json:"email"` // 禁止！
}

// ❌ 依赖外部服务
type User struct {
    repo repository.UserRepository  // 禁止！
}

// ❌ 包含数据库操作
func (u *User) Save() error {
    return db.Create(u).Error  // 禁止！
}
```

## Domain 与 Model 的区别

| 特性 | Domain | Model |
| ---- | ------ | ----- |
| 用途 | 业务概念 | 数据库映射 |
| Tag | 无 | gorm tag |
| 方法 | 业务规则 | 无 |
| 依赖 | 无 | GORM |

```go
// Domain
type User struct {
    ID    string
    Email string
}

// Model
type UserModel struct {
    ID    string `gorm:"primaryKey"`
    Email string `gorm:"uniqueIndex"`
}
```

## 相关文档

- [Model 规范](./08-Model规范.md)
- [项目分层](./02-项目分层.md)
