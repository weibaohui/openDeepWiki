package repository

import (
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetByRepository(repoID uint) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("repository_id = ?", repoID).Order("sort_order").Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) Get(id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) Save(task *model.Task) error {
	return r.db.Save(task).Error
}

// CleanupStuckTasks 清理卡住的running任务（未完成超过指定时间的running任务）
// 用于处理运行中的任务超时
func (r *taskRepository) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := r.db.Model(&model.Task{}).
		Where("status = ? AND started_at < ?", "running", cutoff).
		Updates(map[string]interface{}{
			"status":    "failed",
			"error_msg": fmt.Sprintf("任务超时（超过 %v），已自动标记为失败", timeout),
		})
	return result.RowsAffected, result.Error
}

// CleanupStuckQueuedTasks 清理卡住的queued任务（入队超过指定时间的queued任务）
// 用于处理入队后长时间未执行的任务
func (r *taskRepository) CleanupStuckQueuedTasks(timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := r.db.Model(&model.Task{}).
		Where("status = ? AND updated_at < ?", "queued", cutoff).
		Updates(map[string]interface{}{
			"status":    "failed",
			"error_msg": fmt.Sprintf("任务入队超时（超过 %v），已自动标记为失败", timeout),
		})
	return result.RowsAffected, result.Error
}

func (r *taskRepository) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	cutoff := time.Now().Add(-timeout)
	var tasks []model.Task
	err := r.db.Where("status = ? AND started_at < ?", "running", cutoff).Find(&tasks).Error
	return tasks, err
}

func (r *taskRepository) DeleteByRepositoryID(repoID uint) error {
	return r.db.Where("repository_id = ?", repoID).Delete(&model.Task{}).Error
}

func (r *taskRepository) Delete(id uint) error {
	return r.db.Delete(&model.Task{}, id).Error
}
