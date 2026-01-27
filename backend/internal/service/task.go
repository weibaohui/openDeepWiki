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
)

type TaskService struct {
	cfg        *config.Config
	taskRepo   repository.TaskRepository
	repoRepo   repository.RepoRepository
	docService *DocumentService
}

func NewTaskService(cfg *config.Config, taskRepo repository.TaskRepository, repoRepo repository.RepoRepository, docService *DocumentService) *TaskService {
	return &TaskService{
		cfg:        cfg,
		taskRepo:   taskRepo,
		repoRepo:   repoRepo,
		docService: docService,
	}
}

func (s *TaskService) GetByRepository(repoID uint) ([]model.Task, error) {
	return s.taskRepo.GetByRepository(repoID)
}

func (s *TaskService) Get(id uint) (*model.Task, error) {
	return s.taskRepo.Get(id)
}

func (s *TaskService) Run(taskID uint) error {
	klog.V(6).Infof("开始执行任务: taskID=%d", taskID)

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		klog.V(6).Infof("获取任务失败: taskID=%d, error=%v", taskID, err)
		return err
	}
	klog.V(6).Infof("任务信息: taskID=%d, type=%s, title=%s", task.ID, task.Type, task.Title)

	repo, err := s.repoRepo.GetBasic(task.RepositoryID)
	if err != nil {
		klog.V(6).Infof("获取仓库失败: repoID=%d, error=%v", task.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("仓库信息: repoID=%d, name=%s, localPath=%s", repo.ID, repo.Name, repo.LocalPath)

	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	task.ErrorMsg = ""
	s.taskRepo.Save(task)
	klog.V(6).Infof("任务状态更新为 running: taskID=%d", taskID)

	// 开始执行任务，更新仓库状态为分析中
	s.UpdateRepositoryStatus(task.RepositoryID)

	klog.V(6).Infof("开始静态分析: repoPath=%s", repo.LocalPath)
	projectInfo, err := analyzer.Analyze(repo.LocalPath)
	if err != nil {
		klog.V(6).Infof("静态分析失败: error=%v", err)
		s.failTask(task, fmt.Sprintf("静态分析失败: %v", err))
		return err
	}
	klog.V(6).Infof("静态分析完成: projectType=%s, totalFiles=%d, totalLines=%d",
		projectInfo.Type, projectInfo.TotalFiles, projectInfo.TotalLines)

	klog.V(6).Infof("初始化 LLM 客户端: apiURL=%s, model=%s, maxTokens=%d",
		s.cfg.LLM.APIURL, s.cfg.LLM.Model, s.cfg.LLM.MaxTokens)
	llmClient := llm.NewClient(
		s.cfg.LLM.APIURL,
		s.cfg.LLM.APIKey,
		s.cfg.LLM.Model,
		s.cfg.LLM.MaxTokens,
	)

	llmAnalyzer := analyzer.NewLLMAnalyzer(llmClient)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	klog.V(6).Infof("开始 LLM 分析: taskType=%s", task.Type)
	content, err := llmAnalyzer.Analyze(ctx, analyzer.AnalyzeRequest{
		TaskType:    task.Type,
		ProjectInfo: projectInfo,
	})

	if err != nil {
		klog.V(6).Infof("LLM 分析失败: error=%v", err)
		s.failTask(task, fmt.Sprintf("LLM 分析失败: %v", err))
		return err
	}
	klog.V(6).Infof("LLM 分析完成: contentLength=%d", len(content))

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
		s.failTask(task, fmt.Sprintf("保存文档失败: %v", err))
		return err
	}
	klog.V(6).Infof("文档保存成功")

	completedAt := time.Now()
	task.Status = "completed"
	task.CompletedAt = &completedAt
	s.taskRepo.Save(task)

	duration := completedAt.Sub(now)
	klog.V(6).Infof("任务执行完成: taskID=%d, duration=%v", taskID, duration)

	// 任务完成后更新仓库状态
	s.UpdateRepositoryStatus(task.RepositoryID)

	return nil
}

func (s *TaskService) failTask(task *model.Task, errMsg string) {
	klog.V(6).Infof("任务失败: taskID=%d, error=%s", task.ID, errMsg)
	task.Status = "failed"
	task.ErrorMsg = errMsg
	s.taskRepo.Save(task)

	// 任务失败后更新仓库状态
	s.UpdateRepositoryStatus(task.RepositoryID)
}

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

func (s *TaskService) Reset(taskID uint) error {
	klog.V(6).Infof("重置任务: taskID=%d", taskID)
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return err
	}

	if err := s.docService.DeleteByTaskID(taskID); err != nil {
		return err
	}

	task.Status = "pending"
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil
	err = s.taskRepo.Save(task)
	if err == nil {
		s.UpdateRepositoryStatus(task.RepositoryID)
	}
	return err
}

// ForceReset 强制重置任务，无论当前状态
func (s *TaskService) ForceReset(taskID uint) error {
	klog.V(6).Infof("强制重置任务: taskID=%d", taskID)
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return err
	}

	klog.V(6).Infof("任务当前状态: taskID=%d, status=%s, startedAt=%v",
		taskID, task.Status, task.StartedAt)

	// 删除关联的文档
	if err := s.docService.DeleteByTaskID(taskID); err != nil {
		return err
	}

	// 重置任务状态
	task.Status = "pending"
	task.ErrorMsg = ""
	task.StartedAt = nil
	task.CompletedAt = nil

	klog.V(6).Infof("任务已强制重置: taskID=%d", taskID)
	if err := s.taskRepo.Save(task); err != nil {
		return err
	}
	s.UpdateRepositoryStatus(task.RepositoryID)
	return nil
}

// CleanupStuckTasks 清理卡住的任务（运行超过指定时间的任务）
func (s *TaskService) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	klog.V(6).Infof("开始清理卡住的任务: timeout=%v", timeout)

	affected, err := s.taskRepo.CleanupStuckTasks(timeout)
	if err != nil {
		klog.V(6).Infof("清理卡住任务失败: error=%v", err)
		return 0, err
	}

	klog.V(6).Infof("清理卡住任务完成: affected=%d", affected)
	return affected, nil
}

// GetStuckTasks 获取卡住的任务列表
func (s *TaskService) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return s.taskRepo.GetStuckTasks(timeout)
}

func (s *TaskService) UpdateRepositoryStatus(repoID uint) error {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return err
	}

	// 如果还在克隆中或准备中（尚未开始分析），不自动更新
	if repo.Status == "pending" || repo.Status == "cloning" {
		return nil
	}

	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return err
	}

	var runningCount, pendingCount, failedCount, completedCount int
	for _, t := range tasks {
		switch t.Status {
		case "running":
			runningCount++
		case "pending":
			pendingCount++
		case "failed":
			failedCount++
		case "completed":
			completedCount++
		}
	}

	oldStatus := repo.Status
	if runningCount > 0 {
		repo.Status = "analyzing"
	} else if failedCount > 0 {
		// 如果没有正在运行的任务，且有失败的任务，则状态为 error
		// 但如果还有 pending 的任务，可能还是处于 analyzing 状态（等待继续或手动触发）
		if pendingCount > 0 {
			repo.Status = "analyzing"
		} else {
			repo.Status = "error"
		}
	} else if pendingCount > 0 {
		// 没有运行和失败的任务，但有等待中的任务
		if completedCount > 0 {
			repo.Status = "analyzing"
		} else {
			repo.Status = "ready"
		}
	} else {
		repo.Status = "completed"
	}

	if repo.Status != oldStatus {
		klog.V(6).Infof("更新仓库状态: repoID=%d, old=%s, new=%s", repoID, oldStatus, repo.Status)
		if err := s.repoRepo.Save(repo); err != nil {
			return err
		}
	}
	return nil
}
