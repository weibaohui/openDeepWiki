package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/repository"
	"github.com/opendeepwiki/backend/internal/service/analyzer"
	"github.com/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/opendeepwiki/backend/internal/service/statemachine"
)

type TaskService struct {
	cfg              *config.Config
	taskRepo         repository.TaskRepository
	repoRepo         repository.RepoRepository
	docService       *DocumentService
	taskStateMachine *statemachine.TaskStateMachine
	repoAggregator   *statemachine.RepositoryStatusAggregator
	orchestrator     *orchestrator.Orchestrator
}

func NewTaskService(cfg *config.Config, taskRepo repository.TaskRepository, repoRepo repository.RepoRepository, docService *DocumentService) *TaskService {
	return &TaskService{
		cfg:              cfg,
		taskRepo:         taskRepo,
		repoRepo:         repoRepo,
		docService:       docService,
		taskStateMachine: statemachine.NewTaskStateMachine(),
		repoAggregator:   statemachine.NewRepositoryStatusAggregator(),
	}
}

// SetOrchestrator 设置任务编排器
// 用于解决循环依赖问题
func (s *TaskService) SetOrchestrator(o *orchestrator.Orchestrator) {
	s.orchestrator = o
}

// GetByRepository 获取仓库的所有任务
func (s *TaskService) GetByRepository(repoID uint) ([]model.Task, error) {
	return s.taskRepo.GetByRepository(repoID)
}

// Get 获取单个任务
func (s *TaskService) Get(id uint) (*model.Task, error) {
	return s.taskRepo.Get(id)
}

// Enqueue 提交任务到编排器队列
// 这是新的任务提交方式，通过编排器控制执行
func (s *TaskService) Enqueue(taskID, repositoryID uint, priority int) error {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 状态迁移: pending -> queued
	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusQueued

	// 如果任务已经在队列中，直接入队（用于服务重启恢复或重试），跳过状态迁移
	if oldStatus == statemachine.TaskStatusQueued {
		klog.V(6).Infof("任务已在队列中，重新入队: taskID=%d", taskID)
	} else {
		// 使用状态机验证迁移
		if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
			return fmt.Errorf("任务状态迁移失败: %w", err)
		}

		// 更新数据库状态
		task.Status = string(newStatus)
		if err := s.taskRepo.Save(task); err != nil {
			return fmt.Errorf("更新任务状态失败: %w", err)
		}
	}

	// 提交到编排器
	job := orchestrator.NewTaskJob(taskID, repositoryID, priority)
	if err := s.orchestrator.EnqueueJob(job); err != nil {
		// 入队失败，只有在发生了状态迁移时才回滚状态
		if oldStatus != statemachine.TaskStatusQueued {
			task.Status = string(oldStatus)
			_ = s.taskRepo.Save(task)
		}
		return fmt.Errorf("任务入队失败: %w", err)
	}

	// 更新仓库状态
	_ = s.UpdateRepositoryStatus(repositoryID)

	return nil
}

// Run 执行任务（由编排器调用）
// 注意：这个方法现在由编排器worker调用，不应该直接使用
func (s *TaskService) Run(ctx context.Context, taskID uint) error {
	klog.V(6).Infof("开始执行任务: taskID=%d", taskID)

	// 获取任务
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		klog.V(6).Infof("获取任务失败: taskID=%d, error=%v", taskID, err)
		return err
	}

	// 状态迁移: queued -> running
	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusRunning

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
		return fmt.Errorf("任务状态迁移失败: %w", err)
	}

	// 更新数据库状态
	now := time.Now()
	task.Status = string(newStatus)
	task.StartedAt = &now
	task.ErrorMsg = ""
	if err := s.taskRepo.Save(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	klog.V(6).Infof("任务状态更新为 running: taskID=%d", taskID)

	// 更新仓库状态为分析中
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	// 执行任务逻辑
	execErr := s.executeTaskLogic(ctx, task)

	// 更新最终状态
	if execErr != nil {
		// 任务失败
		_ = s.failTask(task, fmt.Sprintf("任务执行失败: %v", execErr))
		return execErr
	}

	// 任务成功
	_ = s.succeedTask(task)
	return nil
}

// executeTaskLogic 执行任务的核心逻辑
// 不包含状态管理，只负责业务逻辑
func (s *TaskService) executeTaskLogic(ctx context.Context, task *model.Task) error {
	klog.V(6).Infof("任务信息: taskID=%d, type=%s, title=%s", task.ID, task.Type, task.Title)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(task.RepositoryID)
	if err != nil {
		klog.V(6).Infof("获取仓库失败: repoID=%d, error=%v", task.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("仓库信息: repoID=%d, name=%s, localPath=%s", repo.ID, repo.Name, repo.LocalPath)

	// 静态分析
	klog.V(6).Infof("开始静态分析: repoPath=%s", repo.LocalPath)
	projectInfo, err := analyzer.Analyze(repo.LocalPath)
	if err != nil {
		klog.V(6).Infof("静态分析失败: error=%v", err)
		return fmt.Errorf("静态分析失败: %w", err)
	}
	klog.V(6).Infof("静态分析完成: projectType=%s, totalFiles=%d, totalLines=%d",
		projectInfo.Type, projectInfo.TotalFiles, projectInfo.TotalLines)

	// 初始化LLM客户端
	klog.V(6).Infof("初始化 LLM 客户端: apiURL=%s, model=%s, maxTokens=%d",
		s.cfg.LLM.APIURL, s.cfg.LLM.Model, s.cfg.LLM.MaxTokens)
	llmClient := llm.NewClient(
		s.cfg.LLM.APIURL,
		s.cfg.LLM.APIKey,
		s.cfg.LLM.Model,
		s.cfg.LLM.MaxTokens,
	)

	llmAnalyzer := analyzer.NewLLMAnalyzer(llmClient)

	// LLM分析
	klog.V(6).Infof("开始 LLM 分析: taskType=%s", task.Type)
	content, err := llmAnalyzer.Analyze(ctx, analyzer.AnalyzeRequest{
		TaskType:    task.Type,
		ProjectInfo: projectInfo,
	})

	if err != nil {
		klog.V(6).Infof("LLM 分析失败: error=%v", err)
		return fmt.Errorf("LLM 分析失败: %w", err)
	}
	klog.V(6).Infof("LLM 分析完成: contentLength=%d", len(content))

	// 保存文档
	taskDef := getTaskDefinition(task.Type)

	klog.V(6).Infof("保存文档: title=%s, filename=%s", taskDef.Title, taskDef.Filename)
	_, err = s.docService.Create(CreateDocumentRequest{
		RepositoryID: task.RepositoryID,
		TaskID:       task.ID,
		Title:        taskDef.Title,
		Filename:     taskDef.Filename,
		Content:      content,
		SortOrder:    taskDef.SortOrder,
	})

	if err != nil {
		klog.V(6).Infof("保存文档失败: error=%v", err)
		return fmt.Errorf("保存文档失败: %w", err)
	}
	klog.V(6).Infof("文档保存成功")

	return nil
}

// succeedTask 任务成功完成处理
// 状态迁移: running -> succeeded
func (s *TaskService) succeedTask(task *model.Task) error {
	klog.V(6).Infof("任务成功: taskID=%d", task.ID)

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusSucceeded

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
		klog.Errorf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
		return err
	}

	// 更新数据库状态
	completedAt := time.Now()
	task.Status = string(newStatus)
	task.CompletedAt = &completedAt
	if err := s.taskRepo.Save(task); err != nil {
		klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
		return err
	}

	duration := completedAt.Sub(*task.StartedAt)
	klog.V(6).Infof("任务执行完成: taskID=%d, duration=%v", task.ID, duration)

	// 任务完成后更新仓库状态
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

// failTask 任务失败处理
// 状态迁移: running -> failed
func (s *TaskService) failTask(task *model.Task, errMsg string) error {
	klog.V(6).Infof("任务失败: taskID=%d, error=%s", task.ID, errMsg)

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusFailed

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
		klog.Errorf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
		return err
	}

	// 更新数据库状态
	task.Status = string(newStatus)
	task.ErrorMsg = errMsg
	if err := s.taskRepo.Save(task); err != nil {
		klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
		return err
	}

	// 任务失败后更新仓库状态
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

// Reset 重置任务
// 状态迁移: failed/succeeded/canceled -> pending
func (s *TaskService) Reset(taskID uint) error {
	klog.V(6).Infof("重置任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusPending

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
		return fmt.Errorf("任务状态迁移失败: %w", err)
	}

	// 删除关联的文档
	if err := s.docService.DeleteByTaskID(taskID); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}

	// 更新数据库状态
	task.Status = string(newStatus)
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil
	if err := s.taskRepo.Save(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	klog.V(6).Infof("任务已重置: taskID=%d", taskID)

	// 更新仓库状态
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

// ForceReset 强制重置任务，无论当前状态
// 状态迁移: 任意状态 -> pending（除了running）
func (s *TaskService) ForceReset(taskID uint) error {
	klog.V(6).Infof("强制重置任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	klog.V(6).Infof("任务当前状态: taskID=%d, status=%s, startedAt=%v",
		taskID, task.Status, task.StartedAt)

	// 强制重置时，running状态的任务需要特殊处理
	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusPending

	// 如果是running状态，需要先转到canceled，然后再转到pending
	if oldStatus == statemachine.TaskStatusRunning {
		if err := s.taskStateMachine.Transition(oldStatus, statemachine.TaskStatusCanceled, taskID); err != nil {
			klog.Warningf("任务状态迁移失败（running -> canceled）: taskID=%d, error=%v，继续强制重置", taskID, err)
		}
		// 继续执行迁移到pending
	}

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(newStatus, newStatus, taskID); err != nil {
		klog.Warningf("任务状态迁移失败: taskID=%d, error=%v，继续强制重置", taskID, err)
	}

	// 删除关联的文档
	if err := s.docService.DeleteByTaskID(taskID); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}

	// 重置任务状态
	task.Status = string(newStatus)
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil

	klog.V(6).Infof("任务已强制重置: taskID=%d", taskID)
	if err := s.taskRepo.Save(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 更新仓库状态
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

// CleanupStuckTasks 清理卡住的任务（运行超过指定时间的任务）
// 状态迁移: running -> failed (超时)
func (s *TaskService) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	klog.V(6).Infof("开始清理卡住的任务: timeout=%v", timeout)

	// 获取需要清理的任务
	tasks, err := s.taskRepo.GetStuckTasks(timeout)
	if err != nil {
		klog.V(6).Infof("获取卡住任务失败: error=%v", err)
		return 0, err
	}

	var affected int64
	for _, task := range tasks {
		// 状态迁移: running -> failed
		oldStatus := statemachine.TaskStatus(task.Status)
		newStatus := statemachine.TaskStatusFailed

		// 使用状态机验证迁移
		if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
			klog.Warningf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
			continue
		}

		// 更新数据库状态
		task.Status = string(newStatus)
		task.ErrorMsg = fmt.Sprintf("任务超时（超过 %v），已自动标记为失败", timeout)
		if err := s.taskRepo.Save(&task); err != nil {
			klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
			continue
		}

		affected++
		klog.V(6).Infof("清理卡住任务: taskID=%d", task.ID)

		// 更新仓库状态
		_ = s.UpdateRepositoryStatus(task.RepositoryID)
	}

	klog.V(6).Infof("清理卡住任务完成: affected=%d", affected)
	return affected, nil
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskService) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return s.taskRepo.GetStuckTasks(timeout)
}

// UpdateRepositoryStatus 更新仓库状态（使用状态机聚合器）
// 根据任务集合状态推导仓库状态
func (s *TaskService) UpdateRepositoryStatus(repoID uint) error {
	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 获取所有任务
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 构建任务状态汇总
	summary := s.buildTaskSummary(tasks)

	// 使用状态机聚合器计算新状态
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	newStatus, err := s.repoAggregator.AggregateStatus(currentStatus, summary, repoID)
	if err != nil {
		klog.Warningf("仓库状态聚合失败: repoID=%d, error=%v", repoID, err)
		return err
	}

	// 如果状态没有变化，直接返回
	if newStatus == currentStatus {
		return nil
	}

	// 验证状态迁移
	if err := s.repoAggregator.StateMachine.ValidateTransition(currentStatus, newStatus); err != nil {
		klog.Errorf("仓库状态迁移失败: repoID=%d, error=%v", repoID, err)
		return err
	}

	// 更新数据库状态
	repo.Status = string(newStatus)
	if err := s.repoRepo.Save(repo); err != nil {
		return fmt.Errorf("更新仓库状态失败: %w", err)
	}

	klog.V(6).Infof("仓库状态已更新: repoID=%d, %s -> %s", repoID, currentStatus, newStatus)

	return nil
}

// buildTaskSummary 构建任务状态汇总
func (s *TaskService) buildTaskSummary(tasks []model.Task) *statemachine.TaskStatusSummary {
	summary := &statemachine.TaskStatusSummary{
		Total: len(tasks),
	}

	for _, t := range tasks {
		status := statemachine.TaskStatus(t.Status)
		switch status {
		case statemachine.TaskStatusPending:
			summary.Pending++
		case statemachine.TaskStatusQueued:
			summary.Queued++
		case statemachine.TaskStatusRunning:
			summary.Running++
		case statemachine.TaskStatusSucceeded:
			summary.Succeeded++
		case statemachine.TaskStatusFailed:
			summary.Failed++
		case statemachine.TaskStatusCanceled:
			summary.Canceled++
		}
	}

	return summary
}

// getTaskDefinition 获取任务定义
func getTaskDefinition(taskType string) struct {
	Type      string
	Title     string
	Filename  string
	SortOrder int
} {
	for _, t := range model.TaskTypes {
		if t.Type == taskType {
			return t
		}
	}
	return model.TaskTypes[0]
}

// GetOrchestratorStatus 获取编排器状态
func (s *TaskService) GetOrchestratorStatus() *orchestrator.QueueStatus {
	if s.orchestrator == nil {
		return nil
	}
	return s.orchestrator.GetQueueStatus()
}
