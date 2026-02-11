package service

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

// RunAllTasks 执行仓库的所有任务
// 将所有pending任务提交到编排器队列
func (s *RepositoryService) RunAllTasks(repoID uint) error {
	klog.V(6).Infof("准备执行仓库的所有任务: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 检查仓库状态
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return fmt.Errorf("仓库状态不允许执行任务: current=%s", currentStatus)
	}

	// 获取所有任务
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 筛选出pending状态的任务
	var pendingTasks []*model.Task
	for i := range tasks {
		if tasks[i].Status == string(statemachine.TaskStatusPending) {
			pendingTasks = append(pendingTasks, &tasks[i])
		}
	}

	// 如果没有pending任务，直接返回
	if len(pendingTasks) == 0 {
		klog.V(6).Infof("仓库没有待执行的任务: repoID=%d", repoID)
		return nil
	}

	klog.V(6).Infof("找到 %d 个待执行任务: repoID=%d", len(pendingTasks), repoID)

	// 先将所有pending任务状态更新为queued，然后提交到编排器队列
	// 按sort_order顺序处理，保证执行顺序
	for _, task := range pendingTasks {
		// 状态迁移: pending -> queued
		oldStatus := statemachine.TaskStatus(task.Status)
		newStatus := statemachine.TaskStatusQueued

		// 使用状态机验证迁移
		if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
			klog.Errorf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
			return fmt.Errorf("任务状态迁移失败: taskID=%d, %w", task.ID, err)
		}

		// 更新数据库状态
		task.Status = string(newStatus)
		if err := s.taskRepo.Save(task); err != nil {
			klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
			return fmt.Errorf("更新任务状态失败: taskID=%d, %w", task.ID, err)
		}
	}

	// 将所有queued任务提交到编排器队列
	jobs := make([]*orchestrator.Job, 0, len(pendingTasks))
	for _, task := range pendingTasks {
		job := orchestrator.NewTaskJob(task.ID, task.RepositoryID)
		jobs = append(jobs, job)
	}

	// 批量提交到编排器
	if err := s.orchestrator.EnqueueBatch(jobs); err != nil {
		return fmt.Errorf("批量提交任务失败: %w", err)
	}

	klog.V(6).Infof("成功提交 %d 个任务到编排器: repoID=%d", len(jobs), repoID)

	return nil
}
