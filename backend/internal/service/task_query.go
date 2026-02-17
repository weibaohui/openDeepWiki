package service

import (
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
)

// TaskQueryService 任务查询服务
type TaskQueryService struct {
	taskRepo    repository.TaskRepository
	orchestrator *orchestrator.Orchestrator
}

// NewTaskQueryService 创建新的任务查询服务
func NewTaskQueryService(taskRepo repository.TaskRepository) *TaskQueryService {
	return &TaskQueryService{
		taskRepo: taskRepo,
	}
}

// SetOrchestrator 设置编排器
func (s *TaskQueryService) SetOrchestrator(o *orchestrator.Orchestrator) {
	s.orchestrator = o
}

// Get 获取单个任务
func (s *TaskQueryService) Get(id uint) (*model.Task, error) {
	return s.taskRepo.Get(id)
}

// GetByRepository 获取仓库的所有任务
func (s *TaskQueryService) GetByRepository(repoID uint) ([]model.Task, error) {
	return s.taskRepo.GetByRepository(repoID)
}

// GetTaskStats 获取仓库的任务统计信息
func (s *TaskQueryService) GetTaskStats(repoID uint) (map[string]int64, error) {
	return s.taskRepo.GetTaskStats(repoID)
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskQueryService) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return s.taskRepo.GetStuckTasks(timeout)
}

// GetOrchestratorStatus 获取编排器状态
func (s *TaskQueryService) GetOrchestratorStatus() *orchestrator.QueueStatus {
	if s.orchestrator == nil {
		return nil
	}
	return s.orchestrator.GetQueueStatus()
}

// GlobalMonitorData 全局监控数据
type GlobalMonitorData struct {
	QueueStatus *orchestrator.QueueStatus `json:"queue_status"`
	ActiveTasks []model.Task              `json:"active_tasks"`
	RecentTasks []model.Task              `json:"recent_tasks"`
}

// GetGlobalMonitorData 获取全局监控数据
func (s *TaskQueryService) GetGlobalMonitorData() (*GlobalMonitorData, error) {
	status := s.GetOrchestratorStatus()

	activeTasks, err := s.taskRepo.GetActiveTasks()
	if err != nil {
		return nil, err
	}

	recentTasks, err := s.taskRepo.GetRecentTasks(20)
	if err != nil {
		return nil, err
	}

	return &GlobalMonitorData{
		QueueStatus: status,
		ActiveTasks: activeTasks,
		RecentTasks: recentTasks,
	}, nil
}
