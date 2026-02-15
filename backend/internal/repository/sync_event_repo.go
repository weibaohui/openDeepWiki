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
