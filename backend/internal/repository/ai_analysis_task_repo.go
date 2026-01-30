package repository

import (
	"time"

	"github.com/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// AIAnalysisTaskRepository AI分析任务Repository接口
type AIAnalysisTaskRepository interface {
	Create(task *model.AIAnalysisTask) error
	GetByID(id uint) (*model.AIAnalysisTask, error)
	GetByTaskID(taskID string) (*model.AIAnalysisTask, error)
	GetByRepository(repoID uint) (*model.AIAnalysisTask, error)
	GetRunningByRepository(repoID uint) (*model.AIAnalysisTask, error)
	Update(task *model.AIAnalysisTask) error
	Delete(id uint) error
}

// aiAnalysisTaskRepository 实现
type aiAnalysisTaskRepository struct {
	db *gorm.DB
}

// NewAIAnalysisTaskRepository 创建Repository实例
func NewAIAnalysisTaskRepository(db *gorm.DB) AIAnalysisTaskRepository {
	return &aiAnalysisTaskRepository{db: db}
}

// Create 创建任务
func (r *aiAnalysisTaskRepository) Create(task *model.AIAnalysisTask) error {
	return r.db.Create(task).Error
}

// GetByID 根据ID获取任务
func (r *aiAnalysisTaskRepository) GetByID(id uint) (*model.AIAnalysisTask, error) {
	var task model.AIAnalysisTask
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByTaskID 根据TaskID获取任务
func (r *aiAnalysisTaskRepository) GetByTaskID(taskID string) (*model.AIAnalysisTask, error) {
	var task model.AIAnalysisTask
	if err := r.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByRepository 获取仓库的最新分析任务
func (r *aiAnalysisTaskRepository) GetByRepository(repoID uint) (*model.AIAnalysisTask, error) {
	var task model.AIAnalysisTask
	if err := r.db.Where("repository_id = ?", repoID).Order("created_at DESC").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetRunningByRepository 获取仓库正在运行的分析任务
func (r *aiAnalysisTaskRepository) GetRunningByRepository(repoID uint) (*model.AIAnalysisTask, error) {
	var task model.AIAnalysisTask
	if err := r.db.Where("repository_id = ? AND status = ?", repoID, "running").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Update 更新任务
func (r *aiAnalysisTaskRepository) Update(task *model.AIAnalysisTask) error {
	task.UpdatedAt = time.Now()
	return r.db.Save(task).Error
}

// Delete 删除任务
func (r *aiAnalysisTaskRepository) Delete(id uint) error {
	return r.db.Delete(&model.AIAnalysisTask{}, id).Error
}
