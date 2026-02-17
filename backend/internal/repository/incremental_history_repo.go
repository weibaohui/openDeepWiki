package repository

import (
	"context"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type incrementalUpdateHistoryRepository struct {
	db *gorm.DB
}

func NewIncrementalUpdateHistoryRepository(db *gorm.DB) IncrementalUpdateHistoryRepository {
	return &incrementalUpdateHistoryRepository{db: db}
}

func (r *incrementalUpdateHistoryRepository) Create(ctx context.Context, history *model.IncrementalUpdateHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *incrementalUpdateHistoryRepository) ListByRepository(ctx context.Context, repositoryID uint, limit int) ([]model.IncrementalUpdateHistory, error) {
	var items []model.IncrementalUpdateHistory
	tx := r.db.WithContext(ctx).Model(&model.IncrementalUpdateHistory{}).Where("repository_id = ?", repositoryID).
		Order("created_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
