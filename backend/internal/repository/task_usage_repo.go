package repository

import (
	"context"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type taskUsageRepository struct {
	db *gorm.DB
}

// NewTaskUsageRepository 创建 TaskUsage 仓储
func NewTaskUsageRepository(db *gorm.DB) TaskUsageRepository {
	return &taskUsageRepository{db: db}
}

// Create 新增任务用量记录
func (r *taskUsageRepository) Create(ctx context.Context, usage *model.TaskUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}
