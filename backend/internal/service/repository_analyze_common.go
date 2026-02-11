package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type analyzeTaskSpec struct {
	sortOrder    int
	taskTitle    string
	validate     func() error
	generator    func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error)
	afterSuccess func(ctx context.Context, repo *model.Repository, task *model.Task) error
}

// prepareAnalyzeRepository 获取仓库并校验是否允许执行分析任务。
func (s *RepositoryService) prepareAnalyzeRepository(repoID uint, analyzeName string) (*model.Repository, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行%s: current=%s", analyzeName, currentStatus)
	}
	return repo, nil
}

// runAnalyzeTask 创建任务并异步执行分析流程。
func (s *RepositoryService) runAnalyzeTask(ctx context.Context, repoID uint, spec analyzeTaskSpec) (*model.Task, error) {
	if spec.validate != nil {
		if err := spec.validate(); err != nil {
			return nil, err
		}
	}

	klog.V(6).Infof("准备异步分析%s: repoID=%d", spec.taskTitle, repoID)

	repo, err := s.prepareAnalyzeRepository(repoID, spec.taskTitle)
	if err != nil {
		return nil, err
	}

	task, err := s.taskService.CreateTaskWithDoc(ctx, repo.ID, spec.taskTitle, spec.sortOrder)
	if err != nil {
		return nil, fmt.Errorf("创建%s任务失败: %w", spec.taskTitle, err)
	}

	go s.executeAnalyzeTaskAsync(repo, task, spec)

	klog.V(6).Infof("%s已异步启动: repoID=%d, taskID=%d", spec.taskTitle, repoID, task.ID)
	return task, nil
}

// executeAnalyzeTaskAsync 执行分析任务并更新任务与仓库状态。
func (s *RepositoryService) executeAnalyzeTaskAsync(repo *model.Repository, task *model.Task, spec analyzeTaskSpec) {
	klog.V(6).Infof("开始异步%s: repoID=%d, taskID=%d", spec.taskTitle, repo.ID, task.ID)
	startedAt := time.Now()
	clearErrMsg := ""
	taskLabel := spec.taskTitle
	if taskLabel == "" {
		taskLabel = spec.taskTitle
	}
	if err := s.updateTaskStatus(task, statemachine.TaskStatusRunning, &startedAt, nil, &clearErrMsg); err != nil {
		klog.Errorf("更新%s任务状态失败: taskID=%d, error=%v", taskLabel, task.ID, err)
	}

	execCtx := context.Background()
	content, err := spec.generator(execCtx, repo, task)
	if err != nil {
		completedAt := time.Now()

		errMsg := fmt.Sprintf("%s失败: %v", spec.taskTitle, err)
		_ = s.updateTaskStatus(task, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
		s.updateRepositoryStatusAfterTask(repo.ID)
		klog.Errorf("异步%s失败: repoID=%d, taskID=%d, error=%v", spec.taskTitle, repo.ID, task.ID, err)
		return
	}

	_, err = s.docService.Update(task.DocID, content)
	if err != nil {
		completedAt := time.Now()

		errMsg := fmt.Sprintf("保存%s文档失败: %v", spec.taskTitle, err)
		_ = s.updateTaskStatus(task, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
		s.updateRepositoryStatusAfterTask(repo.ID)
		klog.Errorf("保存%s文档失败: repoID=%d, taskID=%d, error=%v", spec.taskTitle, repo.ID, task.ID, err)
		return
	}

	completedAt := time.Now()
	if err := s.updateTaskStatus(task, statemachine.TaskStatusSucceeded, nil, &completedAt, nil); err != nil {
		klog.Errorf("更新%s任务完成状态失败: taskID=%d, error=%v", taskLabel, task.ID, err)
	}
	s.updateRepositoryStatusAfterTask(repo.ID)
	klog.V(6).Infof("异步%s完成: repoID=%d, taskID=%d", spec.taskTitle, repo.ID, task.ID)

	if spec.afterSuccess != nil {
		_ = spec.afterSuccess(execCtx, repo, task)
	}
}
