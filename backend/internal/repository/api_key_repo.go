package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	// List 列出所有启用的配置（按优先级排序）
	List(ctx context.Context) ([]*model.APIKey, error)

	// ListByProvider 按提供商列出配置
	ListByProvider(ctx context.Context, provider string) ([]*model.APIKey, error)

	// ListByNames 按名称列表获取配置（按优先级排序）
	ListByNames(ctx context.Context, names []string) ([]*model.APIKey, error)

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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &apiKey, nil
}

// GetByName 根据名称获取
func (r *apiKeyRepository) GetByName(ctx context.Context, name string) (*model.APIKey, error) {
	var apiKey model.APIKey
	err := r.db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", name).First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
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

// ListByProvider 按提供商列出配置
func (r *apiKeyRepository) ListByProvider(ctx context.Context, provider string) ([]*model.APIKey, error) {
	var apiKeys []*model.APIKey
	err := r.db.WithContext(ctx).
		Where("provider = ? AND deleted_at IS NULL", provider).
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
	type StatsResult struct {
		TotalCount       int64 `json:"total_count"`
		EnabledCount     int64 `json:"enabled_count"`
		DisabledCount    int64 `json:"disabled_count"`
		UnavailableCount int64 `json:"unavailable_count"`
		TotalRequests    int64 `json:"total_requests"`
		TotalErrors      int64 `json:"total_errors"`
	}

	var result StatsResult
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

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_count":        result.TotalCount,
		"enabled_count":      result.EnabledCount,
		"disabled_count":     result.DisabledCount,
		"unavailable_count":  result.UnavailableCount,
		"total_requests":     result.TotalRequests,
		"total_errors":       result.TotalErrors,
	}, nil
}

// ErrAPIKeyNotFound API Key 不存在错误
var ErrAPIKeyNotFound = errors.New("api key not found")

// NameExistsInSlice 检查名称是否在切片中存在
func NameExistsInSlice(ctx context.Context, db *gorm.DB, name string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("name = ? AND deleted_at IS NULL", name).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ErrAPIKeyDuplicate API Key 名称重复错误
var ErrAPIKeyDuplicate = fmt.Errorf("api key name already exists")
