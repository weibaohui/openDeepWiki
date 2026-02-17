package service

import (
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

// TaskCleanupService 任务清理服务
type TaskCleanupService struct {
	taskRepo         repository.TaskRepository
	taskStateMachine *statemachine.TaskStateMachine
	lifecycle        *TaskLifecycleService
}

// NewTaskCleanupService 创建新的任务清理服务
func NewTaskCleanupService(taskRepo repository.TaskRepository, lifecycle *TaskLifecycleService) *TaskCleanupService {
	return &TaskCleanupService{
		taskRepo:         taskRepo,
		taskStateMachine: statemachine.NewTaskStateMachine(),
		lifecycle:        lifecycle,
	}
}

// CleanupStuckTasks 清理卡住的任务（运行超过指定时间的任务）
// 状态迁移: running -> failed (超时)
func (s *TaskCleanupService) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	klog.V(6).Infof("开始清理卡住的任务: timeout=%v", timeout)

	tasks, err := s.taskRepo.GetStuckTasks(timeout)
	if err != nil {
		klog.V(6).Infof("获取卡住任务失败: error=%v", err)
		return 0, err
	}

	var affected int64
	for _, task := range tasks {
		oldStatus := statemachine.TaskStatus(task.Status)
		newStatus := statemachine.TaskStatusFailed

		if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
			klog.Warningf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
			continue
		}

		task.Status = string(newStatus)
		task.ErrorMsg = fmt.Sprintf("任务超时（超过 %v），已自动标记为失败", timeout)
		if err := s.taskRepo.Save(&task); err != nil {
			klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
			continue
		}

		affected++
		klog.V(6).Infof("清理卡住任务: taskID=%d", task.ID)
		_ = s.lifecycle.UpdateRepositoryStatus(task.RepositoryID)
	}

	klog.V(6).Infof("清理卡住任务完成: affected=%d", affected)
	return affected, nil
}

// CleanupQueuedTasksOnStartup 清理启动时遗留的排队任务
func (s *TaskCleanupService) CleanupQueuedTasksOnStartup() (int64, error) {
	klog.V(6).Info("开始清理启动时遗留的排队任务")

	tasks, err := s.taskRepo.GetByStatus(string(statemachine.TaskStatusQueued))
	if err != nil {
		klog.V(6).Infof("获取排队任务失败: error=%v", err)
		return 0, err
	}

	var affected int64
	updatedRepoIDs := make(map[uint]struct{})
	for _, task := range tasks {
		currentStatus := statemachine.TaskStatus(task.Status)
		if err := s.taskStateMachine.Transition(currentStatus, statemachine.TaskStatusCanceled, task.ID); err != nil {
			klog.Warningf("任务状态迁移失败（%s -> canceled）: taskID=%d, error=%v", currentStatus, task.ID, err)
			continue
		}
		currentStatus = statemachine.TaskStatusCanceled
		if err := s.taskStateMachine.Transition(currentStatus, statemachine.TaskStatusPending, task.ID); err != nil {
			klog.Warningf("任务状态迁移失败（%s -> pending）: taskID=%d, error=%v", currentStatus, task.ID, err)
			continue
		}

		task.Status = string(statemachine.TaskStatusPending)
		task.ErrorMsg = ""
		task.StartedAt = nil
		task.CompletedAt = nil
		if err := s.taskRepo.Save(&task); err != nil {
			klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
			continue
		}

		affected++
		updatedRepoIDs[task.RepositoryID] = struct{}{}
		klog.V(6).Infof("启动时清理排队任务完成: taskID=%d", task.ID)
	}

	for repoID := range updatedRepoIDs {
		_ = s.lifecycle.UpdateRepositoryStatus(repoID)
	}

	klog.V(6).Infof("启动时清理排队任务完成: affected=%d", affected)
	return affected, nil
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskCleanupService) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return s.taskRepo.GetStuckTasks(timeout)
}
