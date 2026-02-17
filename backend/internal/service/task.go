package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

// ErrRunAfterNotSatisfied RunAfter依赖未满足错误
var ErrRunAfterNotSatisfied = errors.New("runAfter依赖未满足")

// TaskService 任务服务主入口，协调各子服务
type TaskService struct {
	cfg            *config.Config
	taskRepo       repository.TaskRepository
	repoRepo       repository.RepoRepository
	docService     *DocumentService

	// 子服务
	queryService   *TaskQueryService
	lifecycle      *TaskLifecycleService
	cleanupService *TaskCleanupService

	// 执行相关
	taskStateMachine *statemachine.TaskStateMachine
	orchestrator     *orchestrator.Orchestrator
	writers          []domain.Writer
	schedulerOnce    sync.Once
	taskUsageService TaskUsageService
	bus             *eventbus.TaskEventBus
}

// NewTaskService 创建新的任务服务
func NewTaskService(cfg *config.Config, taskRepo repository.TaskRepository, repoRepo repository.RepoRepository, docService *DocumentService) *TaskService {
	s := &TaskService{
		cfg:              cfg,
		taskRepo:         taskRepo,
		repoRepo:         repoRepo,
		docService:       docService,
		taskStateMachine: statemachine.NewTaskStateMachine(),
		taskUsageService: NewTaskUsageService(repository.NewTaskUsageRepository(nil)),
	}

	// 初始化子服务
	s.queryService = NewTaskQueryService(taskRepo)
	s.lifecycle = NewTaskLifecycleService(taskRepo, repoRepo)
	s.cleanupService = NewTaskCleanupService(taskRepo, s.lifecycle)

	return s
}

// AddWriters 添加写入器
func (s *TaskService) AddWriters(writers ...domain.Writer) {
	for _, w := range writers {
		for _, existing := range s.writers {
			if existing.Name() == w.Name() {
				klog.Errorf("[task.AddWriter] 写入器 %s 已存在", w.Name())
				return
			}
		}
	}
	s.writers = append(s.writers, writers...)
}

// GetWriter 获取写入器
func (s *TaskService) GetWriter(name domain.WriterName) (domain.Writer, error) {
	for _, w := range s.writers {
		if w.Name() == name {
			return w, nil
		}
	}
	return nil, fmt.Errorf("写入器 %s 不存在", name)
}

// SetOrchestrator 设置任务编排器
func (s *TaskService) SetOrchestrator(o *orchestrator.Orchestrator) {
	s.orchestrator = o
	s.queryService.SetOrchestrator(o)
	s.lifecycle.SetOrchestrator(o)
	if o != nil {
		o.SetDependencyChecker(s)
	}
}

// SetEventBus 设置任务事件总线
func (s *TaskService) SetEventBus(bus *eventbus.TaskEventBus) {
	s.bus = bus
	s.lifecycle.SetEventBus(bus)
}

// ==================== 查询方法（委托给 TaskQueryService）====================

// Get 获取单个任务
func (s *TaskService) Get(id uint) (*model.Task, error) {
	return s.queryService.Get(id)
}

// GetByRepository 获取仓库的所有任务
func (s *TaskService) GetByRepository(repoID uint) ([]model.Task, error) {
	return s.queryService.GetByRepository(repoID)
}

// GetTaskStats 获取仓库的任务统计信息
func (s *TaskService) GetTaskStats(repoID uint) (map[string]int64, error) {
	return s.queryService.GetTaskStats(repoID)
}

// GetOrchestratorStatus 获取编排器状态
func (s *TaskService) GetOrchestratorStatus() *orchestrator.QueueStatus {
	return s.queryService.GetOrchestratorStatus()
}

// GetGlobalMonitorData 获取全局监控数据
func (s *TaskService) GetGlobalMonitorData() (*GlobalMonitorData, error) {
	return s.queryService.GetGlobalMonitorData()
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskService) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return s.cleanupService.GetStuckTasks(timeout)
}

// ==================== 入队和执行方法 ====================

// Enqueue 提交任务到编排器队列
func (s *TaskService) Enqueue(taskID uint) error {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}
	allowed, runAfterID, runAfterStatus, err := s.CheckRunAfterSatisfied(taskID)
	if err != nil {
		return fmt.Errorf("RunAfter依赖检查失败: %w", err)
	}
	if !allowed {
		klog.V(6).Infof("RunAfter依赖未满足，任务暂不入队: taskID=%d, runAfter=%d, status=%s", taskID, runAfterID, runAfterStatus)
		return ErrRunAfterNotSatisfied
	}

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusQueued

	if oldStatus == statemachine.TaskStatusQueued {
		klog.V(6).Infof("任务已在队列中，重新入队: taskID=%d", taskID)
		if err := s.taskRepo.Save(task); err != nil {
			return fmt.Errorf("刷新任务时间失败: %w", err)
		}
	} else {
		if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
			return fmt.Errorf("任务状态迁移失败: %w", err)
		}
		task.Status = string(newStatus)
		if err := s.taskRepo.Save(task); err != nil {
			return fmt.Errorf("更新任务状态失败: %w", err)
		}
	}

	job := orchestrator.NewTaskJob(taskID, task.RepositoryID)
	if err := s.orchestrator.EnqueueJob(job); err != nil {
		if oldStatus != statemachine.TaskStatusQueued {
			task.Status = string(oldStatus)
			_ = s.taskRepo.Save(task)
		}
		return fmt.Errorf("任务入队失败: %w", err)
	}

	_ = s.UpdateRepositoryStatus(task.RepositoryID)
	return nil
}

// StartPendingTaskScheduler 启动 Pending 任务定时入队
func (s *TaskService) StartPendingTaskScheduler(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	s.schedulerOnce.Do(func() {
		if s.orchestrator == nil {
			klog.V(6).Infof("编排器未初始化，跳过Pending任务定时入队")
			return
		}
		klog.V(6).Infof("启动Pending任务定时入队: interval=%s", interval)
		ticker := time.NewTicker(interval)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					klog.V(6).Infof("Pending任务定时入队已停止: error=%v", ctx.Err())
					return
				case <-ticker.C:
					s.enqueuePendingTasks(ctx)
				}
			}
		}()
	})
}

func (s *TaskService) enqueuePendingTasks(ctx context.Context) {
	tasks, err := s.taskRepo.GetByStatus(string(statemachine.TaskStatusPending))
	if err != nil {
		klog.V(6).Infof("获取Pending任务失败: error=%v", err)
		return
	}
	if len(tasks) == 0 {
		return
	}
	klog.V(6).Infof("发现Pending任务: count=%d", len(tasks))
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			klog.V(6).Infof("Pending任务入队被中断: error=%v", ctx.Err())
			return
		default:
		}
		if err := s.Enqueue(task.ID); err != nil {
			if errors.Is(err, ErrRunAfterNotSatisfied) {
				continue
			}
			klog.V(6).Infof("Pending任务入队失败: taskID=%d, error=%v", task.ID, err)
			continue
		}
		klog.V(6).Infof("Pending任务已入队: taskID=%d", task.ID)
	}
}

// CheckRunAfterSatisfied 检查任务RunAfter依赖是否满足
func (s *TaskService) CheckRunAfterSatisfied(taskID uint) (bool, uint, string, error) {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return false, 0, "", fmt.Errorf("获取任务失败: %w", err)
	}
	if task.RunAfter == 0 {
		return true, 0, "", nil
	}
	runAfterTask, err := s.taskRepo.Get(task.RunAfter)
	if err != nil {
		return false, task.RunAfter, "", fmt.Errorf("获取RunAfter任务失败: %w", err)
	}
	if runAfterTask.Status == string(statemachine.TaskStatusSucceeded) {
		return true, task.RunAfter, runAfterTask.Status, nil
	}
	return false, task.RunAfter, runAfterTask.Status, nil
}

// Run 执行任务（由编排器调用）
func (s *TaskService) Run(ctx context.Context, taskID uint) error {
	klog.V(6).Infof("开始执行任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		klog.V(6).Infof("获取任务失败: taskID=%d, error=%v", taskID, err)
		return err
	}

	oldStatus := statemachine.TaskStatus(task.Status)
	newStatus := statemachine.TaskStatusRunning

	if err := s.taskStateMachine.Transition(oldStatus, newStatus, taskID); err != nil {
		return fmt.Errorf("任务状态迁移失败: %w", err)
	}

	now := time.Now()
	task.Status = string(newStatus)
	task.StartedAt = &now
	task.ErrorMsg = ""
	if err := s.taskRepo.Save(task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	klog.V(6).Infof("任务状态更新为 running: taskID=%d", taskID)
	_ = s.UpdateRepositoryStatus(task.RepositoryID)

	execErr := s.executeTaskLogic(ctx, task)

	if execErr != nil {
		_ = s.FailTask(task, fmt.Sprintf("任务执行失败: %v", execErr))
		return execErr
	}

	_ = s.SucceedTask(task)
	return nil
}

// executeTaskLogic 执行任务的核心逻辑
func (s *TaskService) executeTaskLogic(ctx context.Context, task *model.Task) error {
	klog.V(6).Infof("任务信息: taskID=%d, title=%s", task.ID, task.Title)

	repo, err := s.repoRepo.GetBasic(task.RepositoryID)
	if err != nil {
		klog.V(6).Infof("获取仓库失败: repoID=%d, error=%v", task.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("仓库信息: repoID=%d, name=%s, localPath=%s", repo.ID, repo.Name, repo.LocalPath)

	writer, err := s.GetWriter(task.WriterName)
	if err != nil {
		klog.Errorf("获取写入器失败: writerName=%s, error=%v", task.WriterName, err)
		return fmt.Errorf("获取写入器失败: %w", err)
	}

	ctx = context.WithValue(ctx, "taskID", task.ID)
	content, err := writer.Generate(ctx, repo.LocalPath, task.Title, task.ID)
	if err != nil {
		klog.Errorf("写入器生成文档失败: writerName=%s, taskTitle=%s, error=%v", task.WriterName, task.Title, err)
		return fmt.Errorf("写入器生成文档失败: %w", err)
	}
	klog.V(6).Infof("文档生成完成: taskTitle=%s, contentLength=%d", task.Title, len(content))

	if task.TaskType == domain.DocWrite {
		_, err = s.docService.Update(task.DocID, content)
		if err != nil {
			klog.V(6).Infof("保存文档失败: error=%v", err)
			return fmt.Errorf("保存文档失败: %w", err)
		}
	} else if task.TaskType == domain.DocRewrite {
		originDoc, err := s.docService.Get(task.DocID)
		if err != nil {
			klog.V(6).Infof("获取原始文档失败: docID=%d, error=%v", task.DocID, err)
			return fmt.Errorf("获取原始文档失败: %w", err)
		}
		newDoc, err := s.docService.Create(CreateDocumentRequest{
			RepositoryID: originDoc.RepositoryID,
			TaskID:       originDoc.TaskID,
			Title:        originDoc.Title,
			Filename:     originDoc.Filename,
			Content:      content,
			SortOrder:    originDoc.SortOrder,
		})
		if err != nil {
			klog.V(6).Infof("创建新版本文档失败: docID=%d, error=%v", task.DocID, err)
			return fmt.Errorf("创建新版本文档失败: %w", err)
		}
		if err := s.docService.TransferLatest(originDoc.ID, newDoc.ID); err != nil {
			klog.V(6).Infof("转移文档版本失败: oldDocID=%d, newDocID=%d, error=%v", originDoc.ID, newDoc.ID, err)
			return fmt.Errorf("转移文档版本失败: %w", err)
		}
		task.DocID = newDoc.ID
		task.UpdatedAt = time.Now()
		if err := s.taskRepo.Save(task); err != nil {
			klog.V(6).Infof("更新任务文档ID失败: taskID=%d, docID=%d, error=%v", task.ID, newDoc.ID, err)
			return fmt.Errorf("更新任务文档ID失败: %w", err)
		}
	}

	return nil
}

// ==================== 生命周期方法（委托给 TaskLifecycleService）====================

// SucceedTask 任务成功完成处理
func (s *TaskService) SucceedTask(task *model.Task) error {
	return s.lifecycle.SucceedTask(task)
}

// FailTask 任务失败处理
func (s *TaskService) FailTask(task *model.Task, errMsg string) error {
	return s.lifecycle.FailTask(task, errMsg)
}

// Reset 重置任务
func (s *TaskService) Reset(taskID uint) error {
	return s.lifecycle.Reset(taskID)
}

// ForceReset 强制重置任务
func (s *TaskService) ForceReset(taskID uint) error {
	return s.lifecycle.ForceReset(taskID)
}

// Retry 重试任务
func (s *TaskService) Retry(taskID uint) error {
	klog.V(6).Infof("重试任务: taskID=%d", taskID)
	if err := s.Reset(taskID); err != nil {
		return fmt.Errorf("重置任务失败: %w", err)
	}
	if err := s.Enqueue(taskID); err != nil {
		return fmt.Errorf("任务入队失败: %w", err)
	}
	return nil
}

// ReGenByNewTask 重新生成任务
func (s *TaskService) ReGenByNewTask(taskID uint) error {
	klog.V(6).Infof("重新生成任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}
	oldDocID := task.DocID
	task, err = s.CreateDocWriteTask(context.Background(), task.RepositoryID, task.Title, task.Outline, task.SortOrder)
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
func (s *TaskService) Cancel(taskID uint) error {
	return s.lifecycle.Cancel(taskID)
}

// Delete 删除任务
func (s *TaskService) Delete(taskID uint) error {
	return s.lifecycle.Delete(taskID, s.docService)
}

// UpdateRepositoryStatus 更新仓库状态
func (s *TaskService) UpdateRepositoryStatus(repoID uint) error {
	return s.lifecycle.UpdateRepositoryStatus(repoID)
}

// ==================== 清理方法（委托给 TaskCleanupService）====================

// CleanupStuckTasks 清理卡住的任务
func (s *TaskService) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	return s.cleanupService.CleanupStuckTasks(timeout)
}

// CleanupQueuedTasksOnStartup 清理启动时遗留的排队任务
func (s *TaskService) CleanupQueuedTasksOnStartup() (int64, error) {
	return s.cleanupService.CleanupQueuedTasksOnStartup()
}

// GetTaskUsageService 获取任务用量服务
func (s *TaskService) GetTaskUsageService() TaskUsageService {
	return s.taskUsageService
}
