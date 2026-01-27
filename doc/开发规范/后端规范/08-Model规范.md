# Model 规范

本文档定义 GORM Model 的编写规范。

## Model 职责

Model 只服务数据库：

- 表结构定义
- GORM tag 声明
- 表名映射

## 标准模板

```go
package model

import "time"

// UserModel 用户表
type UserModel struct {
    ID        string    `gorm:"primaryKey;type:varchar(36)"`
    Email     string    `gorm:"uniqueIndex;type:varchar(255);not null"`
    Name      string    `gorm:"type:varchar(100);not null"`
    Password  string    `gorm:"type:varchar(255);not null"`
    Status    string    `gorm:"type:varchar(20);default:'active'"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (UserModel) TableName() string {
    return "users"
}
```

## 常用 GORM Tag

```go
type ExampleModel struct {
    // 主键
    ID string `gorm:"primaryKey"`

    // 唯一索引
    Email string `gorm:"uniqueIndex"`

    // 普通索引
    Name string `gorm:"index"`

    // 复合索引
    TenantID string `gorm:"index:idx_tenant_user"`
    UserID   string `gorm:"index:idx_tenant_user"`

    // 类型指定
    Content string `gorm:"type:text"`
    Amount  int64  `gorm:"type:bigint"`

    // 默认值
    Status string `gorm:"default:'pending'"`

    // 非空约束
    Title string `gorm:"not null"`

    // 自动时间
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`

    // 软删除
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

## 关联关系

```go
// 一对多
type RepositoryModel struct {
    ID    string `gorm:"primaryKey"`
    Tasks []TaskModel `gorm:"foreignKey:RepositoryID"`
}

type TaskModel struct {
    ID           string `gorm:"primaryKey"`
    RepositoryID string `gorm:"index"`
}

// 多对多
type UserModel struct {
    ID    string `gorm:"primaryKey"`
    Roles []RoleModel `gorm:"many2many:user_roles;"`
}

type RoleModel struct {
    ID    string `gorm:"primaryKey"`
    Users []UserModel `gorm:"many2many:user_roles;"`
}
```

## Model ≠ Domain

```go
// Model - 数据库结构
type UserModel struct {
    ID        string    `gorm:"primaryKey"`
    Email     string    `gorm:"uniqueIndex"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}

// Domain - 业务概念（无 tag）
type User struct {
    ID        string
    Email     string
    CreatedAt time.Time
}
```

## 命名规范

| 类型 | 规范 | 示例 |
| ---- | ---- | ---- |
| Model 名 | XxxModel | UserModel |
| 表名 | 小写复数 | users |
| 字段名 | 驼峰 | CreatedAt |
| 列名 | 蛇形（自动转换） | created_at |

## 禁止事项

```go
// ❌ Model 写业务方法
func (m *UserModel) IsActive() bool {
    return m.Status == "active"  // 禁止！业务方法应在 Domain
}

// ❌ Model 被 Service 直接使用
type userService struct {
    db *gorm.DB  // 禁止！
}

func (s *userService) Get(id string) (*UserModel, error) {
    // 禁止！Service 不应接触 Model
}

// ❌ Model 包含 JSON tag
type UserModel struct {
    ID string `gorm:"primaryKey" json:"id"`  // 禁止混用！
}
```

## 迁移管理

```go
// 仅在初始化脚本中使用 AutoMigrate
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &UserModel{},
        &RepositoryModel{},
        &TaskModel{},
        &DocumentModel{},
    )
}
```

**禁止在运行时调用 AutoMigrate**。

## 相关文档

- [Domain 规范](./06-Domain规范.md)
- [Repository 规范](./07-Repository规范.md)
- [GORM 使用规范](./09-GORM使用规范.md)
