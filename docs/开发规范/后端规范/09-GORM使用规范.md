# GORM 使用规范

本文档定义 GORM 的强制使用规则。

## GORM 使用边界

**GORM 只能出现在 repository 包**，上层永远只依赖 repository interface。

## 强制规则

### 1. 必须使用 WithContext(ctx)

```go
// 正确
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
    // ...
}

// 错误
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.Where("id = ?", id).First(&m).Error  // 禁止！缺少 WithContext
    // ...
}
```

### 2. 每个方法只做一件事

```go
// 正确：单一职责
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) { }
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) { }
func (r *userRepository) ListByStatus(ctx context.Context, status string) ([]*domain.User, error) { }

// 错误：一个方法做太多事
func (r *userRepository) GetUser(ctx context.Context, id, email, status string) (*domain.User, error) {
    query := r.db.WithContext(ctx)
    if id != "" {
        query = query.Where("id = ?", id)
    }
    if email != "" {
        query = query.Where("email = ?", email)
    }
    // ... 动态拼接
}
```

### 3. 显式处理 ErrRecordNotFound

```go
// 正确
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error

    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, repository.ErrNotFound  // 显式转换
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return toDomain(&m), nil
}

// 错误：忽略 ErrRecordNotFound
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
    if err != nil {
        return nil, err  // 没有区分错误类型
    }
    return toDomain(&m), nil
}
```

## 查询模式

### 单条查询

```go
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
    // ...
}
```

### 列表查询

```go
func (r *userRepository) List(ctx context.Context, opts ListOptions) ([]*domain.User, int64, error) {
    var models []model.UserModel
    var total int64

    query := r.db.WithContext(ctx).Model(&model.UserModel{})

    // 条件过滤
    if opts.Status != "" {
        query = query.Where("status = ?", opts.Status)
    }

    // 计数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 分页查询
    if err := query.Offset(opts.Offset).Limit(opts.Limit).Find(&models).Error; err != nil {
        return nil, 0, err
    }

    return toDomainList(models), total, nil
}
```

### 创建

```go
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    m := toModel(user)
    return r.db.WithContext(ctx).Create(m).Error
}
```

### 更新

```go
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
    m := toModel(user)
    return r.db.WithContext(ctx).Save(m).Error
}

// 部分更新
func (r *userRepository) UpdateStatus(ctx context.Context, id string, status string) error {
    return r.db.WithContext(ctx).
        Model(&model.UserModel{}).
        Where("id = ?", id).
        Update("status", status).Error
}
```

### 删除

```go
func (r *userRepository) Delete(ctx context.Context, id string) error {
    return r.db.WithContext(ctx).
        Where("id = ?", id).
        Delete(&model.UserModel{}).Error
}
```

## 禁止事项

```go
// ❌ 链式自由拼查询
func (r *userRepository) Search(ctx context.Context, params map[string]interface{}) {
    query := r.db.WithContext(ctx)
    for k, v := range params {
        query = query.Where(k+" = ?", v)  // 禁止！
    }
}

// ❌ 动态 SQL
func (r *userRepository) Query(ctx context.Context, sql string) {
    r.db.Raw(sql).Scan(&result)  // 禁止！
}

// ❌ 运行时 AutoMigrate
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    r.db.AutoMigrate(&model.UserModel{})  // 禁止！
    return r.db.Create(toModel(user)).Error
}

// ❌ 在 repository 中开启事务
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    tx := r.db.Begin()  // 禁止！
    defer tx.Rollback()
    // ...
    tx.Commit()
}
```

## 日志

使用 klog 记录关键查询：

```go
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    klog.V(6).InfoS("getting user by id", "id", id)
    
    var m model.UserModel
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error

    if err != nil {
        klog.V(2).ErrorS(err, "failed to get user", "id", id)
    }

    return toDomain(&m), nil
}
```

## 相关文档

- [Repository 规范](./07-Repository规范.md)
- [事务规范](./10-事务规范.md)
- [Model 规范](./08-Model规范.md)
