# 017-API Key管理与模型自动切换-设计.md

## 1. 概述

本设计文档详细说明了 API Key 管理与模型自动切换功能的实现方案，包括数据库设计、代码结构、接口定义和核心流程。

### 1.1 模型获取策略（新增）

Agent 获取模型的逻辑如下：
1. **优先使用数据库中的模型**：如果 Agent 指定了模型名称，或者未指定名称但数据库中存在可用的 API Key 配置，优先使用数据库中的模型（按优先级排序）。
2. **Env 环境变量兜底**：如果数据库中没有可用的 API Key 配置（或找不到指定名称的配置），则降级使用环境变量（`env`）中配置的默认模型。
3. **空配置处理**：如果 Agent 的 YAML 定义中未填写 `models` 字段，同样遵守上述规则：尝试从数据库获取优先级最高的可用模型，若失败则使用环境变量兜底。


---

## 2. 架构设计

### 2.1 系统架构图

```mermaid
flowchart TB
    subgraph UI[前端界面]
        ConfigUI[API Key 配置界面]
    end

    subgraph API[API 层]
        Handler[API Key Handler]
    end

    subgraph Service[服务层]
        Service[API Key Service]
    end

    subgraph Repository[数据访问层]
        Repo[API Key Repository]
    end

    subgraph DB[数据库]
        APIKeys[(api_keys 表)]
    end

    subgraph AgentFramework[Agent 框架]
        ModelProvider[Model Provider]
        ModelPool[Model Pool]
        Switcher[Model Switcher]
    end

    subgraph LLM[LLM 调用]
        Client1[Client 1]
        Client2[Client 2]
        Client3[Client 3]
    end

    ConfigUI --> Handler
    Handler --> Service
    Service --> Repo
    Repo --> APIKeys

    Service --> ModelProvider
    ModelProvider --> ModelPool
    ModelProvider --> Switcher
    Switcher --> Client1
    Switcher --> Client2
    Switcher --> Client3
```

### 2.2 模块划分

| 模块名称 | 职责 | 文件位置 |
|---------|------|---------|
| model | 数据模型定义 | `internal/model/api_key.go` |
| repository | 数据库访问 | `internal/repository/api_key_repo.go` |
| service | 业务逻辑 | `internal/service/api_key.go` |
| handler | HTTP 接口 | `internal/handler/api_key.go` |
| model_provider | 模型提供者扩展 | `internal/pkg/adkagents/model_provider.go` |
| model_switcher | 模型切换逻辑 | `internal/pkg/adkagents/model_switcher.go` |

---

## 3. 数据库设计

### 3.1 表结构

```sql
-- api_keys 表
CREATE TABLE api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) UNIQUE NOT NULL,           -- 配置名称
    provider VARCHAR(50) NOT NULL,               -- 服务提供商
    base_url VARCHAR(500) NOT NULL,              -- API 基础 URL
    api_key TEXT NOT NULL,                       -- API Key
    model VARCHAR(255) NOT NULL,                 -- 模型名称
    priority INTEGER DEFAULT 0,                 -- 优先级（越小优先级越高）
    status VARCHAR(20) DEFAULT 'enabled',        -- 状态
    request_count INTEGER DEFAULT 0,             -- 累计请求次数
    error_count INTEGER DEFAULT 0,               -- 累计错误次数
    last_used_at DATETIME,                       -- 最后使用时间
    rate_limit_reset_at DATETIME,               -- 速率限制重置时间
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME                          -- 软删除时间
);

-- 索引
CREATE INDEX idx_api_keys_name ON api_keys(name);
CREATE INDEX idx_api_keys_provider ON api_keys(provider);
CREATE INDEX idx_api_keys_priority ON api_keys(priority);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_deleted_at ON api_keys(deleted_at);
```

### 3.2 数据模型

```go
// internal/model/api_key.go
package model

import (
    "time"
    "gorm.io/gorm"
)

// APIKey API Key 配置
type APIKey struct {
    ID                 uint       `json:"id" gorm:"primaryKey"`
    Name               string     `json:"name" gorm:"size:255;uniqueIndex;not null"`
    Provider           string     `json:"provider" gorm:"size:50;index:idx_api_keys_provider;not null"`
    BaseURL            string     `json:"base_url" gorm:"size:500;not null"`
    APIKey             string     `json:"api_key" gorm:"type:text;not null"`
    Model              string     `json:"model" gorm:"size:255;not null"`
    Priority           int        `json:"priority" gorm:"default:0;index:idx_api_keys_priority"`
    Status             string     `json:"status" gorm:"size:20;default:'enabled';index:idx_api_keys_status"` // enabled/disabled/unavailable
    RequestCount       int        `json:"request_count" gorm:"default:0"`
    ErrorCount         int        `json:"error_count" gorm:"default:0"`
    LastUsedAt         *time.Time `json:"last_used_at"`
    RateLimitResetAt   *time.Time `json:"rate_limit_reset_at"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
    DeletedAt          *time.Time `json:"deleted_at" gorm:"index:idx_api_keys_deleted_at"`
}

// TableName 指定表名
func (APIKey) TableName() string {
    return "api_keys"
}

// MaskAPIKey 脱敏 API Key（只显示前3位和后4位）
func (a *APIKey) MaskAPIKey() string {
    if len(a.APIKey) <= 7 {
        return "***"
    }
    return a.APIKey[:3] + "***" + a.APIKey[len(a.APIKey)-4:]
}

// IsAvailable 检查是否可用
func (a *APIKey) IsAvailable() bool {
    if a.Status != "enabled" {
        return false
    }
    if a.RateLimitResetAt != nil && a.RateLimitResetAt.After(time.Now()) {
        return false
    }
    return true
}

// BeforeUpdate GORM 钩子：更新前自动设置 UpdatedAt
func (a *APIKey) BeforeUpdate(tx *gorm.DB) error {
    a.UpdatedAt = time.Now()
    return nil
}
```

---

## 4. Repository 层设计

### 4.1 接口定义

```go
// internal/repository/api_key_repo.go
package repository

import (
    "context"
    "github.com/weibaohui/opendeepwiki/backend/internal/model"
    "gorm.io/gorm"
)

// APIKeyRepository API Key 仓储接口
type APIKeyRepository interface {
    // Create 创建 API Key 配置
    Create(ctx context.Context, apiKey *model.APIKey) error

    // Update 更新 API Key 配置
    Update(ctx context.Context, apiKey *model.APIKey) error

    // Delete 软删除 API Key 配置
    Delete(ctx context.Context, id uint) error

    // GetByID 根据 ID 获取
    GetByID(ctx context.Context, id uint) (*model.APIKey, error)

    // GetByName 根据名称获取
    GetByName(ctx context.Context, name string) (*model.APIKey, error)

    // List 列出所有配置（按优先级排序，包含已禁用）
    List(ctx context.Context) ([]*model.APIKey, error)

    // ListByProvider 按提供商列出配置
    ListByProvider(ctx context.Context, provider string) ([]*model.APIKey, error)

    // ListByNames 按名称列表获取配置（按优先级排序）
    ListByNames(ctx context.Context, names []string) ([]*model.APIKey, error)

    // GetHighestPriority 获取优先级最高的可用配置
    GetHighestPriority(ctx context.Context) (*model.APIKey, error)

    // UpdateStatus 更新状态
    UpdateStatus(ctx context.Context, id uint, status string) error

    // IncrementStats 增加统计信息
    IncrementStats(ctx context.Context, id uint, requestCount int, errorCount int) error

    // UpdateLastUsedAt 更新最后使用时间
    UpdateLastUsedAt(ctx context.Context, id uint) error

    // SetRateLimitReset 设置速率限制重置时间
    SetRateLimitReset(ctx context.Context, id uint, resetTime time.Time) error

    // GetStats 获取统计信息
    GetStats(ctx context.Context) (map[string]interface{}, error)
}
```

### 4.2 实现示例

```go
// apiKeyRepository API Key 仓储实现
type apiKeyRepository struct {
    db *gorm.DB
}

// NewAPIKeyRepository 创建 API Key 仓储
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
    return &apiKeyRepository{db: db}
}

// Create 创建 API Key 配置
func (r *apiKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
    return r.db.WithContext(ctx).Create(apiKey).Error
}

// Update 更新 API Key 配置
func (r *apiKeyRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
    return r.db.WithContext(ctx).Save(apiKey).Error
}

// Delete 软删除 API Key 配置
func (r *apiKeyRepository) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Delete(&model.APIKey{}, id).Error
}

// GetByID 根据 ID 获取
func (r *apiKeyRepository) GetByID(ctx context.Context, id uint) (*model.APIKey, error) {
    var apiKey model.APIKey
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error
    if err != nil {
        return nil, err
    }
    return &apiKey, nil
}

// GetByName 根据名称获取
func (r *apiKeyRepository) GetByName(ctx context.Context, name string) (*model.APIKey, error) {
    var apiKey model.APIKey
    err := r.db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", name).First(&apiKey).Error
    if err != nil {
        return nil, err
    }
    return &apiKey, nil
}

// List 列出所有配置（按优先级排序，包含已禁用）
func (r *apiKeyRepository) List(ctx context.Context) ([]*model.APIKey, error) {
    var apiKeys []*model.APIKey
    err := r.db.WithContext(ctx).
        Where("deleted_at IS NULL").
        Order("priority ASC, id ASC").
        Find(&apiKeys).Error
    return apiKeys, err
}

// ListByNames 按名称列表获取配置（按优先级排序）
func (r *apiKeyRepository) ListByNames(ctx context.Context, names []string) ([]*model.APIKey, error) {
    if len(names) == 0 {
        return []*model.APIKey{}, nil
    }
    var apiKeys []*model.APIKey
    err := r.db.WithContext(ctx).
        Where("name IN ? AND status = ? AND deleted_at IS NULL", names, "enabled").
        Order("priority ASC, id ASC").
        Find(&apiKeys).Error
    return apiKeys, err
}

// GetHighestPriority 获取优先级最高的可用配置
func (r *apiKeyRepository) GetHighestPriority(ctx context.Context) (*model.APIKey, error) {
    var apiKey model.APIKey
    err := r.db.WithContext(ctx).
        Where("status = ? AND deleted_at IS NULL", "enabled").
        Order("priority ASC, id ASC").
        First(&apiKey).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrAPIKeyNotFound
        }
        return nil, err
    }
    return &apiKey, nil
}

// UpdateStatus 更新状态
func (r *apiKeyRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
    return r.db.WithContext(ctx).
        Model(&model.APIKey{}).
        Where("id = ?", id).
        Update("status", status).Error
}

// IncrementStats 增加统计信息
func (r *apiKeyRepository) IncrementStats(ctx context.Context, id uint, requestCount int, errorCount int) error {
    return r.db.WithContext(ctx).
        Model(&model.APIKey{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "request_count": gorm.Expr("request_count + ?", requestCount),
            "error_count":   gorm.Expr("error_count + ?", errorCount),
        }).Error
}

// UpdateLastUsedAt 更新最后使用时间
func (r *apiKeyRepository) UpdateLastUsedAt(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).
        Model(&model.APIKey{}).
        Where("id = ?", id).
        Update("last_used_at", time.Now()).Error
}

// SetRateLimitReset 设置速率限制重置时间
func (r *apiKeyRepository) SetRateLimitReset(ctx context.Context, id uint, resetTime time.Time) error {
    return r.db.WithContext(ctx).
        Model(&model.APIKey{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "status":             "unavailable",
            "rate_limit_reset_at": resetTime,
        }).Error
}

// GetStats 获取统计信息
func (r *apiKeyRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
    var result struct {
        TotalCount      int64 `json:"total_count"`
        EnabledCount    int64 `json:"enabled_count"`
        DisabledCount   int64 `json:"disabled_count"`
        UnavailableCount int64 `json:"unavailable_count"`
        TotalRequests   int64 `json:"total_requests"`
        TotalErrors     int64 `json:"total_errors"`
    }

    err := r.db.WithContext(ctx).
        Model(&model.APIKey{}).
        Select(`
            COUNT(*) as total_count,
            SUM(CASE WHEN status = 'enabled' THEN 1 ELSE 0 END) as enabled_count,
            SUM(CASE WHEN status = 'disabled' THEN 1 ELSE 0 END) as disabled_count,
            SUM(CASE WHEN status = 'unavailable' THEN 1 ELSE 0 END) as unavailable_count,
            SUM(request_count) as total_requests,
            SUM(error_count) as total_errors
        `).
        Where("deleted_at IS NULL").
        Scan(&result).Error

    return map[string]interface{}{
        "total_count":       result.TotalCount,
        "enabled_count":     result.EnabledCount,
        "disabled_count":    result.DisabledCount,
        "unavailable_count": result.UnavailableCount,
        "total_requests":    result.TotalRequests,
        "total_errors":      result.TotalErrors,
    }, err
}
```

---

## 5. Service 层设计

### 5.1 接口定义

```go
// internal/service/api_key.go
package service

import (
    "context"
    "github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// APIKeyService API Key 服务接口
type APIKeyService interface {
    // CreateAPIKey 创建 API Key 配置
    CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*model.APIKey, error)

    // UpdateAPIKey 更新 API Key 配置
    UpdateAPIKey(ctx context.Context, id uint, req *UpdateAPIKeyRequest) (*model.APIKey, error)

    // DeleteAPIKey 删除 API Key 配置
    DeleteAPIKey(ctx context.Context, id uint) error

    // GetAPIKey 获取 API Key 配置
    GetAPIKey(ctx context.Context, id uint) (*model.APIKey, error)

    // ListAPIKeys 列出所有 API Key 配置
    ListAPIKeys(ctx context.Context) ([]*model.APIKey, error)

    // UpdateAPIKeyStatus 更新状态
    UpdateAPIKeyStatus(ctx context.Context, id uint, status string) error

    // GetStats 获取统计信息
    GetStats(ctx context.Context) (map[string]interface{}, error)

    // RecordRequest 记录请求
    RecordRequest(ctx context.Context, apiKeyID uint, success bool) error

    // MarkUnavailable 标记为不可用
    MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error

    // GetAPIKeyByName 根据名称获取
    GetAPIKeyByName(ctx context.Context, name string) (*model.APIKey, error)

    // GetAPIKeysByNames 根据名称列表获取
    GetAPIKeysByNames(ctx context.Context, names []string) ([]*model.APIKey, error)
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
    Name     string `json:"name" binding:"required"`
    Provider string `json:"provider" binding:"required"`
    BaseURL  string `json:"base_url" binding:"required"`
    APIKey   string `json:"api_key" binding:"required"`
    Model    string `json:"model" binding:"required"`
    Priority int    `json:"priority"`
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
    Name     string `json:"name"`
    Provider string `json:"provider"`
    BaseURL  string `json:"base_url"`
    APIKey   string `json:"api_key"`
    Model    string `json:"model"`
    Priority int    `json:"priority"`
}
```

### 5.2 实现示例

```go
// apiKeyService API Key 服务实现
type apiKeyService struct {
    repo repository.APIKeyRepository
}

// NewAPIKeyService 创建 API Key 服务
func NewAPIKeyService(repo repository.APIKeyRepository) APIKeyService {
    return &apiKeyService{repo: repo}
}

// CreateAPIKey 创建 API Key 配置
func (s *apiKeyService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*model.APIKey, error) {
    // 校验名称唯一性
    existing, err := s.repo.GetByName(ctx, req.Name)
    if err == nil && existing != nil {
        return nil, ErrAPIKeyDuplicate
    }

    apiKey := &model.APIKey{
        Name:     req.Name,
        Provider: req.Provider,
        BaseURL:  req.BaseURL,
        APIKey:   req.APIKey,
        Model:    req.Model,
        Priority: req.Priority,
        Status:   "enabled",
    }

    if err := s.repo.Create(ctx, apiKey); err != nil {
        return nil, err
    }

    return apiKey, nil
}

// UpdateAPIKey 更新 API Key 配置
func (s *apiKeyService) UpdateAPIKey(ctx context.Context, id uint, req *UpdateAPIKeyRequest) (*model.APIKey, error) {
    apiKey, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 更新字段
    if req.Name != "" {
        apiKey.Name = req.Name
    }
    if req.Provider != "" {
        apiKey.Provider = req.Provider
    }
    if req.BaseURL != "" {
        apiKey.BaseURL = req.BaseURL
    }
    if req.APIKey != "" {
        apiKey.APIKey = req.APIKey
    }
    if req.Model != "" {
        apiKey.Model = req.Model
    }
    if req.Priority > 0 {
        apiKey.Priority = req.Priority
    }

    if err := s.repo.Update(ctx, apiKey); err != nil {
        return nil, err
    }

    return apiKey, nil
}

// DeleteAPIKey 删除 API Key 配置
func (s *apiKeyService) DeleteAPIKey(ctx context.Context, id uint) error {
    return s.repo.Delete(ctx, id)
}

// GetAPIKey 获取 API Key 配置
func (s *apiKeyService) GetAPIKey(ctx context.Context, id uint) (*model.APIKey, error) {
    return s.repo.GetByID(ctx, id)
}

// ListAPIKeys 列出所有 API Key 配置
func (s *apiKeyService) ListAPIKeys(ctx context.Context) ([]*model.APIKey, error) {
    return s.repo.List(ctx)
}

// UpdateAPIKeyStatus 更新状态
func (s *apiKeyService) UpdateAPIKeyStatus(ctx context.Context, id uint, status string) error {
    return s.repo.UpdateStatus(ctx, id, status)
}

// GetStats 获取统计信息
func (s *apiKeyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
    return s.repo.GetStats(ctx)
}

// RecordRequest 记录请求
func (s *apiKeyService) RecordRequest(ctx context.Context, apiKeyID uint, success bool) error {
    requestCount := 1
    errorCount := 0
    if !success {
        errorCount = 1
    }
    return s.repo.IncrementStats(ctx, apiKeyID, requestCount, errorCount)
}

// MarkUnavailable 标记为不可用
func (s *apiKeyService) MarkUnavailable(ctx context.Context, apiKeyID uint, resetTime time.Time) error {
    return s.repo.SetRateLimitReset(ctx, apiKeyID, resetTime)
}

// GetAPIKeyByName 根据名称获取
func (s *apiKeyService) GetAPIKeyByName(ctx context.Context, name string) (*model.APIKey, error) {
    return s.repo.GetByName(ctx, name)
}

// GetAPIKeysByNames 根据名称列表获取
func (s *apiKeyService) GetAPIKeysByNames(ctx context.Context, names []string) ([]*model.APIKey, error) {
    return s.repo.ListByNames(ctx, names)
}
```

---

## 6. Handler 层设计

### 6.1 接口定义

```go
// internal/handler/api_key.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "k8s.io/klog/v2"
)

// APIKeyHandler API Key 处理器
type APIKeyHandler struct {
    service service.APIKeyService
}

// NewAPIKeyHandler 创建 API Key 处理器
func NewAPIKeyHandler(service service.APIKeyService) *APIKeyHandler {
    return &APIKeyHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *APIKeyHandler) RegisterRoutes(router *gin.RouterGroup) {
    router.GET("/api-keys", h.ListAPIKeys)
    router.POST("/api-keys", h.CreateAPIKey)
    router.GET("/api-keys/:id", h.GetAPIKey)
    router.PUT("/api-keys/:id", h.UpdateAPIKey)
    router.DELETE("/api-keys/:id", h.DeleteAPIKey)
    router.PATCH("/api-keys/:id/status", h.UpdateStatus)
    router.GET("/api-keys/stats", h.GetStats)
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
    Name     string `json:"name" binding:"required"`
    Provider string `json:"provider" binding:"required"`
    BaseURL  string `json:"base_url" binding:"required"`
    APIKey   string `json:"api_key" binding:"required"`
    Model    string `json:"model" binding:"required"`
    Priority int    `json:"priority"`
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
    Name     string `json:"name"`
    Provider string `json:"provider"`
    BaseURL  string `json:"base_url"`
    APIKey   string `json:"api_key"`
    Model    string `json:"model"`
    Priority int    `json:"priority"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
    Status string `json:"status" binding:"required"` // enabled/disabled
}

// APIKeyResponse API Key 响应（脱敏）
type APIKeyResponse struct {
    ID               uint       `json:"id"`
    Name             string     `json:"name"`
    Provider         string     `json:"provider"`
    BaseURL          string     `json:"base_url"`
    APIKey           string     `json:"api_key"`      // 脱敏后
    Model            string     `json:"model"`
    Priority         int        `json:"priority"`
    Status           string     `json:"status"`
    RequestCount     int        `json:"request_count"`
    ErrorCount       int        `json:"error_count"`
    LastUsedAt       *time.Time `json:"last_used_at"`
    RateLimitResetAt *time.Time `json:"rate_limit_reset_at"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

// CreateAPIKey 创建 API Key 配置
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
    var req CreateAPIKeyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    apiKey, err := h.service.CreateAPIKey(c.Request.Context(), &service.CreateAPIKeyRequest{
        Name:     req.Name,
        Provider: req.Provider,
        BaseURL:  req.BaseURL,
        APIKey:   req.APIKey,
        Model:    req.Model,
        Priority: req.Priority,
    })
    if err != nil {
        klog.Errorf("CreateAPIKey failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, h.toResponse(apiKey))
}

// GetAPIKey 获取 API Key 配置
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
    id := c.Param("id")
    var apiKeyID uint
    if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    apiKey, err := h.service.GetAPIKey(c.Request.Context(), apiKeyID)
    if err != nil {
        klog.Errorf("GetAPIKey failed: %v", err)
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, h.toResponse(apiKey))
}

// ListAPIKeys 列出所有 API Key 配置
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
    apiKeys, err := h.service.ListAPIKeys(c.Request.Context())
    if err != nil {
        klog.Errorf("ListAPIKeys failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    responses := make([]*APIKeyResponse, 0, len(apiKeys))
    for _, apiKey := range apiKeys {
        responses = append(responses, h.toResponse(apiKey))
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  responses,
        "total": len(responses),
    })
}

// UpdateAPIKey 更新 API Key 配置
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
    id := c.Param("id")
    var apiKeyID uint
    if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    var req UpdateAPIKeyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    apiKey, err := h.service.UpdateAPIKey(c.Request.Context(), apiKeyID, &service.UpdateAPIKeyRequest{
        Name:     req.Name,
        Provider: req.Provider,
        BaseURL:  req.BaseURL,
        APIKey:   req.APIKey,
        Model:    req.Model,
        Priority: req.Priority,
    })
    if err != nil {
        klog.Errorf("UpdateAPIKey failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, h.toResponse(apiKey))
}

// DeleteAPIKey 删除 API Key 配置
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
    id := c.Param("id")
    var apiKeyID uint
    if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    if err := h.service.DeleteAPIKey(c.Request.Context(), apiKeyID); err != nil {
        klog.Errorf("DeleteAPIKey failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// UpdateStatus 更新状态
func (h *APIKeyHandler) UpdateStatus(c *gin.Context) {
    id := c.Param("id")
    var apiKeyID uint
    if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    var req UpdateStatusRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.service.UpdateAPIKeyStatus(c.Request.Context(), apiKeyID, req.Status); err != nil {
        klog.Errorf("UpdateStatus failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}

// GetStats 获取统计信息
func (h *APIKeyHandler) GetStats(c *gin.Context) {
    stats, err := h.service.GetStats(c.Request.Context())
    if err != nil {
        klog.Errorf("GetStats failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, stats)
}

// toResponse 转换为响应对象（脱敏 API Key）
func (h *APIKeyHandler) toResponse(apiKey *model.APIKey) *APIKeyResponse {
    return &APIKeyResponse{
        ID:               apiKey.ID,
        Name:             apiKey.Name,
        Provider:         apiKey.Provider,
        BaseURL:          apiKey.BaseURL,
        APIKey:           apiKey.MaskAPIKey(),
        Model:            apiKey.Model,
        Priority:         apiKey.Priority,
        Status:           apiKey.Status,
        RequestCount:     apiKey.RequestCount,
        ErrorCount:       apiKey.ErrorCount,
        LastUsedAt:       apiKey.LastUsedAt,
        RateLimitResetAt: apiKey.RateLimitResetAt,
        CreatedAt:        apiKey.CreatedAt,
        UpdatedAt:        apiKey.UpdatedAt,
    }
}
```

---

## 7. 模型提供者扩展设计

### 7.1 接口扩展

```go
// internal/pkg/adkagents/model_provider.go
package adkagents

import (
    "context"
    "time"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/weibaohui/opendeepwiki/backend/internal/model"
    "github.com/weibaohui/opendeepwiki/backend/internal/repository"
    "k8s.io/klog/v2"
)

// EnhancedModelProvider 增强的模型提供者，支持多模型和自动切换
type EnhancedModelProvider struct {
    config           *config.Config
    apiKeyRepo       repository.APIKeyRepository
    apiKeyService    service.APIKeyService
    defaultModel     model.ToolCallingChatModel
    modelCache       map[string]model.ToolCallingChatModel
    modelCacheMutex  sync.RWMutex
    switcher         *ModelSwitcher
}

// NewEnhancedModelProvider 创建增强的模型提供者
func NewEnhancedModelProvider(
    cfg *config.Config,
    apiKeyRepo repository.APIKeyRepository,
    apiKeyService service.APIKeyService,
    defaultModel model.ToolCallingChatModel,
) *EnhancedModelProvider {
    provider := &EnhancedModelProvider{
        config:        cfg,
        apiKeyRepo:    apiKeyRepo,
        apiKeyService: apiKeyService,
        defaultModel:  defaultModel,
        modelCache:    make(map[string]model.ToolCallingChatModel),
        switcher:      NewModelSwitcher(apiKeyService),
    }
    return provider
}

// GetModel 获取指定名称的模型
func (p *EnhancedModelProvider) GetModel(name string) (model.ToolCallingChatModel, error) {
    // 如果 name 为空，返回默认模型
    if name == "" {
        klog.V(6).Infof("GetModel: using default model")
        return p.defaultModel, nil
    }

    // 检查缓存
    p.modelCacheMutex.RLock()
    if cachedModel, exists := p.modelCache[name]; exists {
        p.modelCacheMutex.RUnlock()
        klog.V(6).Infof("GetModel: using cached model %s", name)
        return cachedModel, nil
    }
    p.modelCacheMutex.RUnlock()

    // 从数据库获取 API Key 配置
    apiKey, err := p.apiKeyRepo.GetByName(context.Background(), name)
    if err != nil {
        klog.Errorf("GetModel: failed to get API Key %s: %v", name, err)
        return p.defaultModel, nil
    }

    // 检查是否可用
    if !apiKey.IsAvailable() {
        klog.Warningf("GetModel: API Key %s is not available (status=%s, rate_limit_reset_at=%v)",
            name, apiKey.Status, apiKey.RateLimitResetAt)
        return nil, ErrModelUnavailable
    }

    // 创建 ChatModel 实例
    chatModel, err := p.createChatModel(apiKey)
    if err != nil {
        klog.Errorf("GetModel: failed to create ChatModel for %s: %v", name, err)
        return nil, err
    }

    // 缓存模型
    p.modelCacheMutex.Lock()
    p.modelCache[name] = chatModel
    p.modelCacheMutex.Unlock()

    klog.V(6).Infof("GetModel: created and cached model %s", name)
    return chatModel, nil
}

// DefaultModel 获取默认模型
func (p *EnhancedModelProvider) DefaultModel() model.ToolCallingChatModel {
    return p.defaultModel
}

// GetModelPool 获取模型池（按优先级排序）
func (p *EnhancedModelProvider) GetModelPool(ctx context.Context, names []string) ([]model.ToolCallingChatModel, error) {
    klog.V(6).Infof("GetModelPool: getting models for names %v", names)

    // 从数据库获取 API Key 配置列表
    apiKeys, err := p.apiKeyRepo.ListByNames(ctx, names)
    if err != nil {
        klog.Errorf("GetModelPool: failed to get API Keys: %v", err)
        return nil, err
    }

    // 过滤可用的配置并创建模型
    models := make([]model.ToolCallingChatModel, 0, len(apiKeys))
    for _, apiKey := range apiKeys {
        if !apiKey.IsAvailable() {
            klog.V(6).Infof("GetModelPool: skipping unavailable model %s", apiKey.Name)
            continue
        }

        // 创建 ChatModel 实例
        chatModel, err := p.createChatModel(apiKey)
        if err != nil {
            klog.Errorf("GetModelPool: failed to create ChatModel for %s: %v", apiKey.Name, err)
            continue
        }

        models = append(models, chatModel)
    }

    klog.V(6).Infof("GetModelPool: got %d available models", len(models))
    return models, nil
}

// createChatModel 创建 ChatModel 实例
func (p *EnhancedModelProvider) createChatModel(apiKey *model.APIKey) (model.ToolCallingChatModel, error) {
    config := &openai.ChatModelConfig{
        BaseURL:   apiKey.BaseURL,
        APIKey:    apiKey.APIKey,
        Model:     apiKey.Model,
        MaxTokens: &p.config.LLM.MaxTokens,
    }

    chatModel, err := openai.NewChatModel(context.Background(), config)
    if err != nil {
        return nil, err
    }

    // 包装模型，添加 API Key ID 以便跟踪
    return &ModelWithMetadata{
        ChatModel: chatModel,
        APIKeyName: apiKey.Name,
        APIKeyID:   apiKey.ID,
    }, nil
}

// IsRateLimitError 判断错误是否为 Rate Limit 错误
func (p *EnhancedModelProvider) IsRateLimitError(err error) bool {
    if err == nil {
        return false
    }

    errMsg := err.Error()

    // 检查 HTTP 状态码
    if strings.Contains(errMsg, "429") {
        return true
    }

    // 检查错误消息
    rateLimitKeywords := []string{
        "rate limit",
        "rate_limit",
        "quota exceeded",
        "too many requests",
        "rate-limited",
        "request rate exceeded",
    }

    lowerMsg := strings.ToLower(errMsg)
    for _, keyword := range rateLimitKeywords {
        if strings.Contains(lowerMsg, keyword) {
            return true
        }
    }

    return false
}

// MarkModelUnavailable 标记模型为不可用
func (p *EnhancedModelProvider) MarkModelUnavailable(modelName string, resetTime time.Time) error {
    ctx := context.Background()

    // 获取 API Key 配置
    apiKey, err := p.apiKeyRepo.GetByName(ctx, modelName)
    if err != nil {
        return err
    }

    // 标记为不可用
    err = p.apiKeyService.MarkUnavailable(ctx, apiKey.ID, resetTime)
    if err != nil {
        return err
    }

    // 清除缓存
    p.modelCacheMutex.Lock()
    delete(p.modelCache, modelName)
    p.modelCacheMutex.Unlock()

    klog.Warningf("MarkModelUnavailable: marked model %s as unavailable, reset at %v", modelName, resetTime)
    return nil
}

// GetNextModel 获取下一个可用模型
func (p *EnhancedModelProvider) GetNextModel(currentModelName string, poolNames []string) (model.ToolCallingChatModel, error) {
    ctx := context.Background()

    // 获取模型池
    models, err := p.GetModelPool(ctx, poolNames)
    if err != nil {
        return nil, err
    }

    if len(models) == 0 {
        return nil, ErrNoAvailableModel
    }

    // 找到当前模型的位置
    currentIndex := -1
    for i, model := range models {
        if modelWithMeta, ok := model.(*ModelWithMetadata); ok {
            if modelWithMeta.APIKeyName == currentModelName {
                currentIndex = i
                break
            }
        }
    }

    // 如果当前模型不在池中，返回第一个可用模型
    if currentIndex == -1 {
        klog.V(6).Infof("GetNextModel: current model not in pool, returning first model")
        return models[0], nil
    }

    // 返回下一个模型
    if currentIndex+1 < len(models) {
        nextModel := models[currentIndex+1]
        klog.V(6).Infof("GetNextModel: switching from index %d to %d", currentIndex, currentIndex+1)
        return nextModel, nil
    }

    // 没有下一个模型
    return nil, ErrNoAvailableModel
}
```

---

## 8. 模型切换逻辑设计

### 8.1 ModelWithMetadata 结构

```go
// internal/pkg/adkagents/model_switcher.go
package adkagents

import (
    "context"
    "time"
    "github.com/cloudwego/eino/components/model"
    "github.com/weibaohui/opendeepwiki/backend/internal/service"
    "k8s.io/klog/v2"
)

// ModelWithMetadata 带有元数据的模型包装器
type ModelWithMetadata struct {
    model.ToolCallingChatModel
    APIKeyName string
    APIKeyID   uint
}

// Name 返回模型名称
func (m *ModelWithMetadata) Name() string {
    return m.APIKeyName
}

// ModelSwitcher 模型切换器
type ModelSwitcher struct {
    apiKeyService    service.APIKeyService
    modelProvider    ModelProvider
}

// NewModelSwitcher 创建模型切换器
func NewModelSwitcher(apiKeyService service.APIKeyService) *ModelSwitcher {
    return &ModelSwitcher{
        apiKeyService: apiKeyService,
    }
}

// CallWithRetry 使用模型切换重试机制调用
func (s *ModelSwitcher) CallWithRetry(
    ctx context.Context,
    provider ModelProvider,
    poolNames []string,
    fn func(model.ToolCallingChatModel) (interface{}, error),
) (interface{}, error) {
    maxRetries := 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        klog.V(6).Infof("CallWithRetry: attempt %d/%d", attempt+1, maxRetries)

        // 获取模型池
        models, err := provider.GetModelPool(ctx, poolNames)
        if err != nil {
            klog.Errorf("CallWithRetry: failed to get model pool: %v", err)
            return nil, err
        }

        if len(models) == 0 {
            klog.Error("CallWithRetry: no available models")
            return nil, ErrNoAvailableModel
        }

        // 使用第一个可用模型
        currentModel := models[0]
        if modelWithMeta, ok := currentModel.(*ModelWithMetadata); ok {
            klog.V(6).Infof("CallWithRetry: using model %s", modelWithMeta.APIKeyName)
        }

        // 调用函数
        result, err := fn(currentModel)
        if err != nil {
            klog.V(6).Infof("CallWithRetry: error occurred: %v", err)

            // 检查是否为 Rate Limit 错误
            if provider.IsRateLimitError(err) {
                // 获取模型名称
                var modelName string
                if modelWithMeta, ok := currentModel.(*ModelWithMetadata); ok {
                    modelName = modelWithMeta.APIKeyName
                }

                klog.Warningf("CallWithRetry: rate limit hit for model %s", modelName)

                // 标记当前模型为不可用
                resetTime := s.parseResetTime(err)
                if resetTime.IsZero() {
                    // 如果没有明确的重置时间，设置默认为 1 小时后
                    resetTime = time.Now().Add(time.Hour)
                }

                if modelName != "" {
                    provider.MarkModelUnavailable(modelName, resetTime)
                }

                // 如果还有重试机会，继续下一次尝试
                if attempt+1 < maxRetries {
                    klog.Infof("CallWithRetry: retrying with next model...")
                    time.Sleep(1 * time.Second) // 等待 1 秒后重试
                    continue
                }

                return nil, ErrAllModelsUnavailable
            }

            // 非 Rate Limit 错误，直接返回
            return nil, err
        }

        // 成功，记录使用情况
        if modelWithMeta, ok := currentModel.(*ModelWithMetadata); ok {
            s.apiKeyService.RecordRequest(ctx, modelWithMeta.APIKeyID, true)
        }

        return result, nil
    }

    return nil, ErrMaxRetriesExceeded
}

// parseResetTime 从错误中解析重置时间
func (s *ModelSwitcher) parseResetTime(err error) time.Time {
    errMsg := err.Error()

    // 尝试从错误消息中解析重置时间
    // 格式可能为：Try again in 60s, Retry after 1m, Reset at 2026-02-04 12:00:00
    patterns := []string{
        `Try again in (\d+)s`,
        `Retry after (\d+)s`,
        `Try again in (\d+)m`,
        `Retry after (\d+)m`,
    }

    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(errMsg)
        if len(matches) >= 2 {
            var duration int
            if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
                if strings.Contains(pattern, "m") {
                    return time.Now().Add(time.Duration(duration) * time.Minute)
                }
                return time.Now().Add(time.Duration(duration) * time.Second)
            }
        }
    }

    // 无法解析，返回零值
    return time.Time{}
}
```

---

## 9. Agent 配置解析扩展

### 9.1 AgentDefinition 扩展

```go
// internal/pkg/adkagents/agent.go
package adkagents

import (
    "time"
)

// AgentDefinition ADK Agent 定义（从 YAML 加载）
type AgentDefinition struct {
    // 元数据
    Name        string `yaml:"name" json:"name"`
    Description string `yaml:"description" json:"description"`

    // LLM 配置（支持单模型和多模型）
    Model       string   `yaml:"model" json:"model"`       // 单模型：模型名称或别名
    Models      []string `yaml:"models" json:"models"`      // 多模型：模型列表

    // Agent 行为配置
    Instruction   string   `yaml:"instruction" json:"instruction"`
    Tools         []string `yaml:"tools" json:"tools"`
    MaxIterations int      `yaml:"maxIterations" json:"max_iterations"`

    // 可选配置
    Exit ExitConfig `yaml:"exit,omitempty" json:"exit,omitempty"`

    // 路径信息（运行时填充）
    Path     string    `json:"path"`
    LoadedAt time.Time `json:"loaded_at"`
}

// GetModelNames 获取模型名称列表
func (a *AgentDefinition) GetModelNames() []string {
    // 如果配置了多模型列表，返回列表
    if len(a.Models) > 0 {
        return a.Models
    }

    // 如果配置了单模型，返回包含单模型的列表
    if a.Model != "" {
        return []string{a.Model}
    }

    // 都没有配置，返回空列表（使用默认模型）
    return []string{}
}

// UseModelPool 判断是否使用模型池
func (a *AgentDefinition) UseModelPool() bool {
    return len(a.Models) > 0
}
```

### 9.2 Manager 扩展

```go
// internal/pkg/adkagents/manager.go
package adkagents

import (
    "context"
    "fmt"
    "sync"
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/model"
    "k8s.io/klog/v2"
)

// Manager ADK Agent 管理器
type Manager struct {
    config         *Config
    registry       *Registry
    modelProvider  ModelProvider
    cache          map[string]adk.Agent
    cacheMutex     sync.RWMutex
}

// NewManager 创建 Manager
func NewManager(config *Config, modelProvider ModelProvider) (*Manager, error) {
    manager := &Manager{
        config:        config,
        registry:      NewRegistry(),
        modelProvider: modelProvider,
        cache:         make(map[string]adk.Agent),
    }

    // 加载所有 Agent
    if err := manager.LoadAllAgents(); err != nil {
        return nil, err
    }

    return manager, nil
}

// GetAgent 获取指定名称的 ADK Agent 实例
func (m *Manager) GetAgent(name string) (adk.Agent, error) {
    // 检查缓存
    m.cacheMutex.RLock()
    if cachedAgent, exists := m.cache[name]; exists {
        m.cacheMutex.RUnlock()
        return cachedAgent, nil
    }
    m.cacheMutex.RUnlock()

    // 从注册表获取 Agent 定义
    agentDef, err := m.registry.Get(name)
    if err != nil {
        return nil, err
    }

    // 创建 Agent
    agent, err := m.createAgent(agentDef)
    if err != nil {
        return nil, err
    }

    // 缓存 Agent
    m.cacheMutex.Lock()
    m.cache[name] = agent
    m.cacheMutex.Unlock()

    return agent, nil
}

// createAgent 创建 Agent 实例
func (m *Manager) createAgent(agentDef *AgentDefinition) (adk.Agent, error) {
    ctx := context.Background()

    // 获取模型
    var model model.ToolCallingChatModel
    var err error

    if agentDef.UseModelPool() {
        // 使用模型池
        modelNames := agentDef.GetModelNames()
        models, err := m.modelProvider.GetModelPool(ctx, modelNames)
        if err != nil || len(models) == 0 {
            klog.Warningf("createAgent: failed to get model pool for agent %s, using default model", agentDef.Name)
            model = m.modelProvider.DefaultModel()
        } else {
            // 包装模型池
            model = &ModelPool{
                Models:        models,
                ModelProvider: m.modelProvider,
                ModelNames:    modelNames,
            }
        }
    } else {
        // 使用单个模型
        modelName := agentDef.Model
        model, err = m.modelProvider.GetModel(modelName)
        if err != nil {
            klog.Warningf("createAgent: failed to get model %s for agent %s, using default model", modelName, agentDef.Name)
            model = m.modelProvider.DefaultModel()
        }
    }

    // 创建 ChatModelAgent
    cfg := &adk.ChatModelAgentConfig{
        Name:        agentDef.Name,
        Description: agentDef.Description,
        ChatModel:   model,
        Instruction: agentDef.Instruction,
    }

    agent, err := adk.NewChatModelAgent(ctx, cfg)
    if err != nil {
        return nil, err
    }

    return agent, nil
}

// ModelPool 模型池包装器
type ModelPool struct {
    Models        []model.ToolCallingChatModel
    ModelProvider ModelProvider
    ModelNames    []string
    current       int
}

// Generate 实现模型生成接口，支持自动切换
func (p *ModelPool) Generate(ctx context.Context, messages []model.Message, opts ...model.Option) (model.Message, error) {
    maxRetries := len(p.Models)

    for attempt := 0; attempt < maxRetries; attempt++ {
        if p.current >= len(p.Models) {
            p.current = 0
        }

        currentModel := p.Models[p.current]

        // 调用模型
        result, err := currentModel.Generate(ctx, messages, opts...)
        if err != nil {
            // 检查是否为 Rate Limit 错误
            if p.ModelProvider.IsRateLimitError(err) {
                klog.Warningf("ModelPool: rate limit hit, switching to next model")

                // 标记当前模型为不可用
                if modelWithMeta, ok := currentModel.(*ModelWithMetadata); ok {
                    p.ModelProvider.MarkModelUnavailable(modelWithMeta.APIKeyName, time.Now().Add(time.Hour))
                }

                // 切换到下一个模型
                p.current++
                continue
            }

            // 非 Rate Limit 错误，直接返回
            return result, err
        }

        // 成功，返回结果
        return result, nil
    }

    return nil, ErrAllModelsUnavailable
}
```

---

## 10. 数据库迁移

### 10.1 AutoMigrate 更新

```go
// internal/pkg/database/database.go
package database

import (
    "github.com/glebarez/sqlite"
    "github.com/weibaohui/opendeepwiki/backend/internal/model"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func InitDB(dbType, dsn string) (*gorm.DB, error) {
    var dialector gorm.Dialector

    switch dbType {
    case "mysql":
        dialector = mysql.Open(dsn)
    default:
        dialector = sqlite.Open(dsn)
    }

    db, err := gorm.Open(dialector, &gorm.Config{})
    if err != nil {
        return nil, err
    }

    if err := db.AutoMigrate(&model.Repository{}, &model.Task{}, &model.Document{}); err != nil {
        return nil, err
    }
    if err := db.AutoMigrate(&model.DocumentTemplate{}, &model.TemplateChapter{}, &model.TemplateDocument{}); err != nil {
        return nil, err
    }
    if err := db.AutoMigrate(&model.AIAnalysisTask{}); err != nil {
        return nil, err
    }
    // 新增：API Key 表迁移
    if err := db.AutoMigrate(&model.APIKey{}); err != nil {
        return nil, err
    }
    return db, nil
}
```

---

## 11. 路由注册

### 11.1 Router 更新

```go
// internal/router/router.go
package router

import (
    "github.com/gin-gonic/gin"
    "github.com/weibaohui/opendeepwiki/backend/internal/handler"
)

func SetupRoutes(
    r *gin.Engine,
    apiKeyHandler *handler.APIKeyHandler,
    // 其他 handlers...
) {
    api := r.Group("/api")
    {
        // API Key 管理
        apiKeyHandler.RegisterRoutes(api)

        // 其他路由...
    }
}
```

---

## 12. 测试设计

### 12.1 单元测试

```go
// internal/repository/api_key_repo_test.go
package repository_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/weibaohui/opendeepwiki/backend/internal/model"
    "github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

func TestAPIKeyRepository_Create(t *testing.T) {
    // 测试创建 API Key
}

func TestAPIKeyRepository_GetByName(t *testing.T) {
    // 测试根据名称获取
}

func TestAPIKeyRepository_ListByNames(t *testing.T) {
    // 测试按名称列表获取
}

func TestAPIKeyRepository_UpdateStatus(t *testing.T) {
    // 测试更新状态
}

func TestAPIKeyRepository_SetRateLimitReset(t *testing.T) {
    // 测试设置速率限制重置时间
}

func TestAPIKeyRepository_GetStats(t *testing.T) {
    // 测试获取统计信息
}
```

### 12.2 集成测试

```go
// internal/service/api_key_test.go
package service_test

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/weibaohui/opendeepwiki/backend/internal/service"
)

func TestAPIKeyService_CreateAPIKey(t *testing.T) {
    // 测试创建 API Key
}

func TestAPIKeyService_MarkUnavailable(t *testing.T) {
    // 测试标记为不可用
}

func TestAPIKeyService_GetAPIKeysByNames(t *testing.T) {
    // 测试根据名称列表获取
}
```

### 12.3 模型切换测试

```go
// internal/pkg/adkagents/model_switcher_test.go
package adkagents_test

import (
    "context"
    "errors"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
)

func TestModelSwitcher_CallWithRetry(t *testing.T) {
    // 测试模型切换重试逻辑
}

func TestModelSwitcher_ParseResetTime(t *testing.T) {
    // 测试重置时间解析
}

func TestEnhancedModelProvider_IsRateLimitError(t *testing.T) {
    // 测试 Rate Limit 错误检测
}

func TestEnhancedModelProvider_GetModelPool(t *testing.T) {
    // 测试模型池获取
}
```

---

## 13. 文件结构

```
backend/
├── internal/
│   ├── model/
│   │   └── api_key.go                          # API Key 数据模型
│   ├── repository/
│   │   └── api_key_repo.go                    # API Key 仓储
│   ├── service/
│   │   └── api_key.go                         # API Key 服务
│   ├── handler/
│   │   └── api_key.go                         # API Key Handler
│   └── pkg/
│       └── adkagents/
│           ├── model_provider.go                # 增强的模型提供者
│           ├── model_switcher.go               # 模型切换器
│           ├── agent.go                         # AgentDefinition 扩展
│           └── manager.go                       # Manager 扩展
├── internal/pkg/database/
│   └── database.go                             # 数据库迁移更新
└── internal/router/
    └── router.go                               # 路由注册更新
```

---

## 14. 安全考虑

1. **API Key 脱敏**：在 API 响应中，API Key 需要脱敏显示
2. **权限控制**：API Key 管理接口需要添加权限控制
3. **日志安全**：日志中不记录完整的 API Key
4. **传输安全**：API Key 通过 HTTPS 传输

---

## 15. 性能优化

1. **模型缓存**：已创建的模型实例进行缓存，避免重复创建
2. **索引优化**：数据库查询使用合适的索引
3. **连接池**：数据库使用连接池
4. **异步处理**：统计数据更新可以异步处理

---

## 16. 实施计划

### 阶段 1：数据库和模型层
- [ ] 创建 `api_keys` 表
- [ ] 实现 `model.APIKey` 模型
- [ ] 实现 `repository.APIKeyRepository`

### 阶段 2：服务层
- [ ] 实现 `service.APIKeyService`
- [ ] 实现 `handler.APIKeyHandler`
- [ ] 注册路由

### 阶段 3：模型提供者扩展
- [ ] 实现 `EnhancedModelProvider`
- [ ] 实现 `ModelSwitcher`
- [ ] 扩展 `AgentDefinition`
- [ ] 扩展 `Manager`

### 阶段 4：测试和集成
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 端到端测试

### 阶段 5：文档和部署
- [ ] 编写使用文档
- [ ] 更新配置示例
- [ ] 部署验证
