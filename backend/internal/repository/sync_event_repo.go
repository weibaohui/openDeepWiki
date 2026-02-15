package repository

import (
	"context"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type syncEventRepository struct {
	db *gorm.DB
}

func NewSyncEventRepository(db *gorm.DB) SyncEventRepository {
	return &syncEventRepository{db: db}
}

func (r *syncEventRepository) Create(ctx context.Context, event *model.SyncEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// List 查询同步事件列表
func (r *syncEventRepository) List(ctx context.Context, repositoryID uint, eventTypes []string, limit int) ([]model.SyncEvent, error) {
	var events []model.SyncEvent
	tx := r.db.WithContext(ctx).Model(&model.SyncEvent{})
	if repositoryID > 0 {
		tx = tx.Where("repository_id = ?", repositoryID)
	}
	if len(eventTypes) > 0 {
		tx = tx.Where("event_type IN ?", eventTypes)
	}
	tx = tx.Order("created_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
