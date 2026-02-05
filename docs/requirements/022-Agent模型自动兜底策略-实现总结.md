# 022-Agent模型自动兜底策略-实现总结

## 1. 功能概述

实现了 Agent 获取模型的自动兜底策略。在 Agent 未明确指定模型，或指定模型不可用时，系统将按照以下优先级自动选择模型：

1.  **数据库中的模型**：优先使用数据库中配置的、状态为 `enabled` 的、优先级最高（`priority` 值最小）的 API Key 配置。
2.  **Env 环境变量兜底**：如果数据库中没有任何可用的 API Key 配置，则降级使用环境变量（`env`）中配置的默认模型。

这一策略确保了 Agent 在无需硬编码模型名称的情况下，能够动态使用系统中配置的最佳模型，并在配置缺失时具备基本的可用性。

## 2. 需求对应关系

| 需求点 | 实现情况 | 说明 |
| :--- | :--- | :--- |
| **优先使用数据库模型** | ✅ 已实现 | 当请求模型名称为空时，优先调用 `apiKeyRepo.GetHighestPriority` 获取数据库中最佳模型 |
| **Env 环境变量兜底** | ✅ 已实现 | 如果数据库无可用模型，自动回退到 `defaultModel` (Env 配置) |
| **空配置处理** | ✅ 已实现 | Agent YAML 未填写 `models` 字段时（传入空名称），触发上述自动选择逻辑 |

## 3. 关键实现点

### 3.1 数据库层 (`Repository`)

在 `APIKeyRepository` 接口及实现中新增了 `GetHighestPriority` 方法：

```go
// GetHighestPriority 获取优先级最高的可用配置
func (r *apiKeyRepository) GetHighestPriority(ctx context.Context) (*model.APIKey, error) {
    var apiKey model.APIKey
    err := r.db.WithContext(ctx).
        Where("status = ? AND deleted_at IS NULL", "enabled").
        Order("priority ASC, id ASC").
        First(&apiKey).Error
    // ...
    return &apiKey, nil
}
```

### 3.2 模型提供层 (`ModelProvider`)

修改了 `EnhancedModelProviderImpl.GetModel` 方法的核心逻辑：

```go
func (p *EnhancedModelProviderImpl) GetModel(name string) (*openai.ChatModel, error) {
    // 如果 name 为空，尝试使用数据库中的最高优先级模型
    if name == "" {
        apiKey, err := p.apiKeyRepo.GetHighestPriority(context.Background())
        if err == nil && apiKey != nil {
            // 递归调用，使用查到的具体模型名
            return p.GetModel(apiKey.Name)
        }
        // 数据库无可用模型，使用默认模型（Env配置）
        return &p.defaultModel.ChatModel, nil
    }
    // ... 后续逻辑不变（查缓存、查数据库指定名称）
}
```

### 3.3 Agent 管理层 (`Manager`)

修复了 `Manager.createADKAgent` 中当 Agent 未指定模型时的处理逻辑，确保调用 `EnhancedModelProvider` 进行自动兜底，而不是直接使用 Env 默认模型。

```go
// internal/pkg/adkagents/manager.go
} else {
    // 模型未指定，尝试使用增强提供者获取自动兜底模型
    if m.enhancedModelProvider != nil {
        model, err := m.enhancedModelProvider.GetModel("")
        // ... 成功则使用 model，失败则降级到 Env
    } else {
        // 没有增强提供者，直接使用默认模型
        chatModel, err = NewLLMChatModel(m.cfg)
        // ...
    }
}
```

## 4. 验证结果

- **单元测试**: `internal/repository/api_key_repo_test.go` 中新增了对 `GetHighestPriority` 的测试，验证了按优先级排序和状态过滤的正确性。
- **逻辑验证**:
    - 场景1：数据库有 enabled 的 Key -> `GetModel("")` 返回该 Key 对应的模型。
    - 场景2：数据库无 enabled 的 Key -> `GetModel("")` 返回 Env 配置的默认模型。
