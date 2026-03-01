package repository

import (
	"context"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type embeddingKeyRepository struct {
	db *gorm.DB
}

func NewEmbeddingKeyRepository(db *gorm.DB) EmbeddingKeyRepository {
	return &embeddingKeyRepository{db: db}
}

// Create 创建嵌入模型配置
func (r *embeddingKeyRepository) Create(ctx context.Context, key *model.EmbeddingKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

// GetByID 根据ID获取配置
func (r *embeddingKeyRepository) GetByID(ctx context.Context, id uint) (*model.EmbeddingKey, error) {
	var key model.EmbeddingKey
	err := r.db.WithContext(ctx).First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// List 列出所有配置
func (r *embeddingKeyRepository) List(ctx context.Context) ([]model.EmbeddingKey, error) {
	var keys []model.EmbeddingKey
	err := r.db.WithContext(ctx).Order("priority DESC, id ASC").Find(&keys).Error
	return keys, err
}

// GetAvailable 获取可用的配置（按优先级排序）
func (r *embeddingKeyRepository) GetAvailable(ctx context.Context) ([]model.EmbeddingKey, error) {
	var keys []model.EmbeddingKey
	err := r.db.WithContext(ctx).
		Where("status = ?", "enabled").
		Where("deleted_at IS NULL").
		Order("priority DESC, id ASC").
		Find(&keys).Error
	return keys, err
}

// Update 更新配置
func (r *embeddingKeyRepository) Update(ctx context.Context, key *model.EmbeddingKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}

// Delete 删除配置
func (r *embeddingKeyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.EmbeddingKey{}, id).Error
}

// IncrementRequestCount 增加请求计数
func (r *embeddingKeyRepository) IncrementRequestCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbeddingKey{}).
		Where("id = ?", id).
		UpdateColumn("request_count", gorm.Expr("request_count + ?", 1)).
		Error
}

// IncrementErrorCount 增加错误计数
func (r *embeddingKeyRepository) IncrementErrorCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbeddingKey{}).
		Where("id = ?", id).
		UpdateColumn("error_count", gorm.Expr("error_count + 1")).
		Error
}

// UpdateLastUsedAt 更新最后使用时间
func (r *embeddingKeyRepository) UpdateLastUsedAt(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmbeddingKey{}).
		Where("id = ?", id).
		Update("last_used_at", &now).
		Error
}

// SetStatus 设置状态
func (r *embeddingKeyRepository) SetStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbeddingKey{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}