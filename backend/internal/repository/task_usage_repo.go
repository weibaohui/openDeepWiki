package repository

import (
	"context"
	"errors"

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

// GetByTaskID 根据 task_id 查询任务用量记录
// 返回最新的记录（如果有多条）
func (r *taskUsageRepository) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
	var usage model.TaskUsage
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("id DESC").
		First(&usage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没有记录返回 nil
		}
		return nil, err
	}
	return &usage, nil
}

// GetByTaskIDList 根据 task_id 查询任务用量记录列表
func (r *taskUsageRepository) GetByTaskIDList(ctx context.Context, taskID uint) ([]model.TaskUsage, error) {
	var usages []model.TaskUsage
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("id ASC").
		Find(&usages).Error
	if err != nil {
		return nil, err
	}
	return usages, nil
}

// Upsert 根据 task_id 插入或更新任务用量记录
// 如果存在该 task_id 的记录，则删除旧记录并插入新记录（覆盖逻辑）
func (r *taskUsageRepository) Upsert(ctx context.Context, usage *model.TaskUsage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除该 task_id 的所有旧记录
		if err := tx.Where("task_id = ?", usage.TaskID).Delete(&model.TaskUsage{}).Error; err != nil {
			return err
		}
		// 插入新记录
		return tx.Create(usage).Error
	})
}

// UpsertMany 覆盖写入同一 task_id 的用量记录
func (r *taskUsageRepository) UpsertMany(ctx context.Context, usages []model.TaskUsage) error {
	if len(usages) == 0 {
		return nil
	}
	taskID := usages[0].TaskID
	for _, usage := range usages {
		if usage.TaskID != taskID {
			return errors.New("task_id 不一致")
		}
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", taskID).Delete(&model.TaskUsage{}).Error; err != nil {
			return err
		}
		return tx.Create(&usages).Error
	})
}
