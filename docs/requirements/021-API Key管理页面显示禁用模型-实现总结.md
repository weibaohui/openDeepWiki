# 021-API Key管理页面显示禁用模型-实现总结

## 1. 功能概述

修复了 API Key 管理页面无法显示“已禁用”状态模型的问题。此前，后端 `List` 接口仅返回状态为 `enabled` 的模型，导致用户在前端禁用某个模型后，该模型从列表中消失，无法再次启用或管理。

本次修改确保管理接口返回所有状态的 API Key 配置，由前端负责展示状态（已启用/已禁用/不可用）。

## 2. 需求对应关系

| 需求点 | 实现情况 | 说明 |
| :--- | :--- | :--- |
| **显示所有模型** | ✅ 已实现 | 后端 `List` 接口移除状态过滤，返回所有配置 |
| **保持模型切换逻辑** | ✅ 已实现 | 模型切换逻辑使用的 `ListByNames` 接口保持只返回 `enabled` 模型 |

## 3. 关键实现点

### 3.1 后端修改
- **文件**: `backend/internal/repository/api_key_repo.go`
- **修改**: 修改 `List` 方法，移除了 `Where("status = ? ...", "enabled")` 条件，改为只过滤已删除记录 (`deleted_at IS NULL`)。

```go
// List 列出所有配置（按优先级排序，包含已禁用）
func (r *apiKeyRepository) List(ctx context.Context) ([]*model.APIKey, error) {
    var apiKeys []*model.APIKey
    err := r.db.WithContext(ctx).
        Where("deleted_at IS NULL").
        Order("priority ASC, id ASC").
        Find(&apiKeys).Error
    return apiKeys, err
}
```

### 3.2 单元测试
- **新增文件**: `backend/internal/repository/api_key_repo_test.go`
- **内容**: 
    - `TestAPIKeyRepository_List`: 验证 `List` 方法返回所有状态（enabled, disabled, unavailable）的记录。
    - 验证 `ListByNames` 方法仅返回 enabled 状态的记录，确保不影响模型自动切换逻辑。

## 4. 验证结果

- **编译**: 通过。
- **测试**: `go test -v internal/repository/api_key_repo_test.go internal/repository/api_key_repo.go` (需在 backend 目录下运行) 通过。
- **前端表现**: 前端 `APIKeyManager` 组件已具备处理 `disabled` 状态的逻辑（通过 Switch 组件显示），无需修改前端代码即可正常展示和操作已禁用的模型。
