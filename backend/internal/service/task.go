package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/documentgenerator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

type TaskService struct {
	cfg              *config.Config
	taskRepo         repository.TaskRepository
	repoRepo         repository.RepoRepository
	docService       *DocumentService
	docGenerator     *documentgenerator.Service
	taskStateMachine *statemachine.TaskStateMachine
	repoAggregator   *statemachine.RepositoryStatusAggregator
	orchestrator     *orchestrator.Orchestrator
	llm              einomodel.ToolCallingChatModel
}

func NewTaskService(cfg *config.Config, taskRepo repository.TaskRepository, repoRepo repository.RepoRepository, docService *DocumentService, docGenerator *documentgenerator.Service) *TaskService {
	llm, err := adkagents.NewLLMChatModel(cfg)
	if err != nil {
		klog.Errorf("[task.New] 创建LLM失败: %v", err)
	}

	return &TaskService{
		cfg:              cfg,
		taskRepo:         taskRepo,
		repoRepo:         repoRepo,
		docService:       docService,
		docGenerator:     docGenerator,
		taskStateMachine: statemachine.NewTaskStateMachine(),
		repoAggregator:   statemachine.NewRepositoryStatusAggregator(),
		llm:              llm,
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

// GetTaskStats 获取仓库的任务统计信息
func (s *TaskService) GetTaskStats(repoID uint) (map[string]int64, error) {
	return s.taskRepo.GetTaskStats(repoID)
}

// Get 获取单个任务
func (s *TaskService) Get(id uint) (*model.Task, error) {
	return s.taskRepo.Get(id)
}

// Enqueue 提交任务到编排器队列
// 这是新的任务提交方式，通过编排器控制执行
func (s *TaskService) Enqueue(taskID, repositoryID uint) error {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 验证 repositoryID 是否匹配
	if task.RepositoryID != repositoryID {
		return fmt.Errorf("repository ID mismatch: expected %d, got %d", task.RepositoryID, repositoryID)
	}

	// 状态迁移: pending -> queued
	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusQueued

	// 如果任务已经在队列中，直接入队（用于服务重启恢复或重试），跳过状态迁移
	if oldStatus == statemachine.TaskStatusQueued {
		klog.V(6).Infof("任务已在队列中，重新入队: taskID=%d", taskID)
		// 刷新任务更新时间
		if err := s.taskRepo.Save(task); err != nil {
			return fmt.Errorf("刷新任务时间失败: %w", err)
		}
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
	job := orchestrator.NewTaskJob(taskID, task.RepositoryID)
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
	klog.V(6).Infof("任务信息: taskID=%d, title=%s", task.ID, task.Title)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(task.RepositoryID)
	if err != nil {
		klog.V(6).Infof("获取仓库失败: repoID=%d, error=%v", task.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("仓库信息: repoID=%d, name=%s, localPath=%s", repo.ID, repo.Name, repo.LocalPath)

	// 使用 DocumentGeneratorService 生成文档
	klog.V(6).Infof("开始生成文档:  repoPath=%s", repo.LocalPath)
	content, err := s.docGenerator.Generate(ctx, repo.LocalPath, task.Title, task.ID)
	if err != nil {
		klog.Errorf("生成文档失败: taskTitle=%s, error=%v", task.Title, err)
		return fmt.Errorf("生成文档失败: %w", err)
	}
	klog.V(6).Infof("文档生成完成: taskTitle=%s, contentLength=%d", task.Title, len(content))

	_, err = s.docService.Update(task.DocID, content)

	if err != nil {
		klog.V(6).Infof("保存文档失败: error=%v", err)
		return fmt.Errorf("保存文档失败: %w", err)
	}
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

	// 如果是running或queued状态，需要先转到canceled，然后再转到pending
	currentStatus := oldStatus
	if currentStatus == statemachine.TaskStatusRunning || currentStatus == statemachine.TaskStatusQueued {
		if err := s.taskStateMachine.Transition(currentStatus, statemachine.TaskStatusCanceled, taskID); err != nil {
			klog.Warningf("任务状态迁移失败（%s -> canceled）: taskID=%d, error=%v，继续强制重置", currentStatus, taskID, err)
		} else {
			currentStatus = statemachine.TaskStatusCanceled
		}
	}

	// 使用状态机验证迁移到pending
	if currentStatus != newStatus {
		if err := s.taskStateMachine.Transition(currentStatus, newStatus, taskID); err != nil {
			klog.Warningf("任务状态迁移失败（%s -> %s）: taskID=%d, error=%v，继续强制重置", currentStatus, newStatus, taskID, err)
		}
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

// Retry 重试任务
// 组合 Reset 和 Enqueue 操作
// 适用于 Failed, Succeeded, Canceled 状态的任务
func (s *TaskService) Retry(taskID uint) error {
	klog.V(6).Infof("重试任务: taskID=%d", taskID)

	// 1. 先重置任务状态
	if err := s.Reset(taskID); err != nil {
		return fmt.Errorf("重置任务失败: %w", err)
	}

	// 重新获取任务以获取 RepositoryID
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 2. 重新入队
	// 默认优先级为0
	if err := s.Enqueue(taskID, task.RepositoryID); err != nil {
		return fmt.Errorf("任务入队失败: %w", err)
	}

	return nil
}

// CreateTaskWithDoc 创建文档和任务，并建立双向关联
// 1. 创建文档
// 2. 创建任务
// 3. 更新文档关联的任务ID
func (s *TaskService) CreateTaskWithDoc(ctx context.Context, repoID uint, title string, sortOrder int) (*model.Task, error) {
	doc, err := s.docService.Create(CreateDocumentRequest{
		RepositoryID: repoID,
		Title:        title,
		Filename:     title + ".md",
		Content:      "",
		SortOrder:    sortOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("创建文档失败: %w", err)
	}

	task := &model.Task{
		RepositoryID: repoID,
		DocID:        doc.ID,
		Title:        title,
		Status:       string(statemachine.TaskStatusPending),
		SortOrder:    sortOrder,
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	if err := s.docService.UpdateTaskID(doc.ID, task.ID); err != nil {
		// 记录日志但不返回错误，因为任务和文档已创建
		klog.Errorf("更新文档关联的任务ID失败: docID=%d, taskID=%d, error=%v", doc.ID, task.ID, err)
	}

	return task, nil
}

// ReGenByNewTask 重新生成任务
func (s *TaskService) ReGenByNewTask(taskID uint) error {
	klog.V(6).Infof("重新生成任务: taskID=%d", taskID)

	// 重新获取任务以获取 RepositoryID
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}
	oldDocID := task.DocID
	task, err = s.CreateTaskWithDoc(context.Background(), task.RepositoryID, task.Title, task.SortOrder)
	if err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	err = s.docService.TransferLatest(oldDocID, task.DocID)
	if err != nil {
		return fmt.Errorf("删除最新文档失败: %w", err)
	}

	return nil
}

// Cancel 取消任务
// 支持取消 Running 和 Queued 状态的任务
func (s *TaskService) Cancel(taskID uint) error {
	klog.V(6).Infof("取消任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusCanceled

	// 检查是否已经是取消状态
	if oldStatus == newStatus {
		return nil
	}

	// 尝试通知编排器取消正在运行的任务
	// 即使任务不在运行中（例如在队列中），我们也继续更新数据库状态
	// worker在取出任务执行时，应该检查数据库状态（目前worker逻辑依赖外部调用CancelTask来终止上下文）
	// TODO: 最好在worker执行前增加一次状态检查
	if oldStatus == statemachine.TaskStatusRunning {
		if s.orchestrator.CancelTask(taskID, task.RepositoryID) {
			klog.V(6).Infof("已触发运行中任务的取消: taskID=%d", taskID)
		} else {
			klog.Warningf("尝试取消运行中任务，但编排器中未找到: taskID=%d", taskID)
		}
	}

	// 使用状态机验证迁移
	if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
		return fmt.Errorf("任务状态迁移失败: %w", err)
	}

	// 更新数据库状态
	// 记录取消时间为完成时间
	now := time.Now()
	task.Status = string(newStatus)
	task.CompletedAt = &now
	task.ErrorMsg = "用户手动取消"

	if err := s.taskRepo.Save(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	klog.V(6).Infof("任务已取消: taskID=%d", taskID)

	// 更新仓库状态
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

// Delete 删除任务（删除单个任务）
// 注意：删除任务也会删除关联的文档
func (s *TaskService) Delete(taskID uint) error {
	klog.V(6).Infof("删除任务: taskID=%d", taskID)

	// 获取任务
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	repoID := task.RepositoryID

	// 检查任务状态，运行中的任务不允许删除
	currentStatus := statemachine.TaskStatus(task.Status)
	if currentStatus == statemachine.TaskStatusRunning || currentStatus == statemachine.TaskStatusQueued {
		return fmt.Errorf("运行中或已入队的任务不允许删除: current=%s", currentStatus)
	}

	// 删除关联的文档
	if err := s.docService.DeleteByTaskID(taskID); err != nil {
		return fmt.Errorf("删除关联文档失败: %w", err)
	}

	// 删除任务
	if err := s.taskRepo.Delete(taskID); err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}

	klog.V(6).Infof("任务已删除: taskID=%d", taskID)

	// 更新仓库状态
	_ = s.UpdateRepositoryStatus(repoID)

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

// CleanupQueuedTasksOnStartup 清理启动时遗留的排队任务
func (s *TaskService) CleanupQueuedTasksOnStartup() (int64, error) {
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
		_ = s.UpdateRepositoryStatus(repoID)
	}

	klog.V(6).Infof("启动时清理排队任务完成: affected=%d", affected)
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

// GetOrchestratorStatus 获取编排器状态
func (s *TaskService) GetOrchestratorStatus() *orchestrator.QueueStatus {
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
func (s *TaskService) GetGlobalMonitorData() (*GlobalMonitorData, error) {
	status := s.GetOrchestratorStatus()

	activeTasks, err := s.taskRepo.GetActiveTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}

	recentTasks, err := s.taskRepo.GetRecentTasks(20) // 获取最近20条历史记录
	if err != nil {
		return nil, fmt.Errorf("failed to get recent tasks: %w", err)
	}

	return &GlobalMonitorData{
		QueueStatus: status,
		ActiveTasks: activeTasks,
		RecentTasks: recentTasks,
	}, nil
}

// GenerateUserRequestTitle 使用LLM生成用户需求的标题
func (s *TaskService) GenerateUserRequestTitle(ctx context.Context, userContent string) (string, error) {
	if s.llm == nil {
		return "", fmt.Errorf("LLM未初始化")
	}

	klog.V(6).Infof("[task.GenerateUserRequestTitle] 开始生成标题: 用户内容=%s", userContent)

	// 构建Prompt
	prompt := fmt.Sprintf(`你是一个文档标题生成专家。请根据用户的输入，提取核心内容并生成一个简洁、专业的文档标题。

要求：
1. 标题长度不超过 50 个字符
2. 标题应该清晰、专业、易于理解
3. 使用中文或英文（根据用户输入语言）
4. 只返回标题，不要返回其他内容

用户输入：%s

请生成文档标题：`, userContent)

	// 调用LLM
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个专业的文档标题生成专家，擅长从用户需求中提取核心内容并生成简洁的标题。",
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	resp, err := s.llm.Generate(ctx, messages)
	if err != nil {
		klog.Errorf("[task.GenerateUserRequestTitle] LLM调用失败: error=%v", err)
		return "", fmt.Errorf("LLM调用失败: %w", err)
	}

	// 提取响应内容
	title := ""
	if len(resp.Content) > 0 {
		title = strings.TrimSpace(resp.Content)
	}

	// 截断标题，确保不超过50个字符
	if len(title) > 50 {
		title = title[:50]
		klog.V(6).Infof("[task.GenerateUserRequestTitle] 标题过长，已截断: 原标题=%s, 截断后=%s", title, title[:50])
		title = title[:50]
	}

	if title == "" {
		return "", fmt.Errorf("LLM生成标题为空")
	}

	klog.V(6).Infof("[task.GenerateUserRequestTitle] 标题生成成功: 标题=%s", title)
	return title, nil
}
