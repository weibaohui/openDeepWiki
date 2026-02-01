package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/repository"
	"github.com/opendeepwiki/backend/internal/service/einodoc/adk"
	"github.com/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

// AIAnalyzeService AI分析服务
type AIAnalyzeService struct {
	cfg            *config.Config
	repoRepo       repository.RepoRepository
	taskRepo       repository.AIAnalysisTaskRepository
	einoDocService *adk.ADKRepoDocService
}

// NewAIAnalyzeService 创建AI分析服务
func NewAIAnalyzeService(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.AIAnalysisTaskRepository) *AIAnalyzeService {

	// 创建 Eino RepoDoc Service
	einoAdkDocService, err := adk.NewADKRepoDocService(cfg, cfg.Data.RepoDir)
	if err != nil {
		klog.Errorf("创建 Eino RepoDoc Service 失败: %v", err)
		// 如果创建失败，使用 nil，后续会处理错误
	}
	// // 创建 Eino RepoDoc Service
	// einoDocService, err := einodoc.NewEinoRepoDocService(cfg.Data.RepoDir, llmCfg)
	// if err != nil {
	// 	klog.Errorf("创建 Eino RepoDoc Service 失败: %v", err)
	// 	// 如果创建失败，使用 nil，后续会处理错误
	// }

	return &AIAnalyzeService{
		cfg:            cfg,
		repoRepo:       repoRepo,
		taskRepo:       taskRepo,
		einoDocService: einoAdkDocService,
	}
}

// StartAnalysisRequest 启动分析请求
type StartAnalysisRequest struct {
	RepositoryID uint `json:"repository_id" binding:"required"`
}

// StartAnalysisResponse 启动分析响应
type StartAnalysisResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// StartAnalysis 启动AI分析
func (s *AIAnalyzeService) StartAnalysis(repoID uint) (*StartAnalysisResponse, error) {
	klog.V(6).Infof("启动AI分析: repoID=%d", repoID)

	// 1. 获取仓库信息
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	// 2. 检查仓库状态
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许分析: current=%s", currentStatus)
	}

	// 3. 检查是否已有进行中的分析任务
	existingTask, _ := s.taskRepo.GetRunningByRepository(repoID)
	if existingTask != nil {
		klog.V(6).Infof("仓库已有运行中的分析任务: repoID=%d, taskID=%s", repoID, existingTask.TaskID)
		return &StartAnalysisResponse{
			TaskID:  existingTask.TaskID,
			Status:  existingTask.Status,
			Message: "分析任务已在进行中",
		}, nil
	}

	// 4. 生成任务ID
	taskID := uuid.New().String()

	// 5. 创建输出目录
	outputDir := filepath.Join(repo.LocalPath, ".opendeepwiki")
	outputPath := filepath.Join(outputDir, "analysis-report.md")

	// 6. 创建分析任务记录
	task := &model.AIAnalysisTask{
		RepositoryID: repoID,
		TaskID:       taskID,
		Status:       "pending",
		Progress:     0,
		OutputPath:   outputPath,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("创建分析任务失败: %w", err)
	}

	klog.V(6).Infof("AI分析任务创建成功: repoID=%d, taskID=%s", repoID, taskID)

	// 7. 异步执行分析
	go s.executeAnalysis(task, repo)

	return &StartAnalysisResponse{
		TaskID:  taskID,
		Status:  "started",
		Message: "AI分析已启动",
	}, nil
}

// GetAnalysisStatus 获取分析状态
func (s *AIAnalyzeService) GetAnalysisStatus(repoID uint) (*model.AIAnalysisTask, error) {
	// 获取最新的分析任务
	task, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取分析状态失败: %w", err)
	}
	return task, nil
}

// executeAnalysis 执行AI分析（异步）
func (s *AIAnalyzeService) executeAnalysis(task *model.AIAnalysisTask, repo *model.Repository) {
	klog.V(6).Infof("开始执行AI分析: taskID=%s, repoPath=%s", task.TaskID, repo.LocalPath)

	// 1. 更新状态为 running
	task.Status = "running"
	task.Progress = 10
	if err := s.taskRepo.Update(task); err != nil {
		klog.Errorf("更新任务状态失败: taskID=%s, error=%v", task.TaskID, err)
	}

	// 2. 检查仓库路径
	if _, err := os.Stat(repo.LocalPath); err != nil {
		s.failTask(task, fmt.Sprintf("仓库路径不存在: %v", err))
		return
	}

	// 3. 检查 Eino Service 是否可用
	if s.einoDocService == nil {
		s.failTask(task, "Eino Doc Service 未初始化")
		return
	}

	// 4. 使用 Eino RepoDoc Service 执行分析
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	task.Progress = 30
	s.taskRepo.Update(task)

	klog.V(6).Infof("调用 Eino RepoDoc Service: repoPath=%s", repo.LocalPath)

	result, err := s.einoDocService.ParseRepo(ctx, repo.LocalPath)
	if err != nil {
		klog.Errorf("Eino RepoDoc Service 执行失败: taskID=%s, error=%v", task.TaskID, err)
		s.failTask(task, err.Error())
		return
	}

	task.Progress = 80
	s.taskRepo.Update(task)

	// 5. 保存分析结果到文件
	outputDir := filepath.Dir(task.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		s.failTask(task, fmt.Sprintf("创建输出目录失败: %v", err))
		return
	}

	// 组装完整的分析报告
	var reportContent string
	reportContent += fmt.Sprintf("# AI Analysis Report for %s\n\n", repo.Name)
	reportContent += fmt.Sprintf("**Repository:** %s\n\n", repo.URL)
	reportContent += fmt.Sprintf("**Type:** %s\n\n", result.RepoType)
	reportContent += fmt.Sprintf("**Tech Stack:** %v\n\n", result.TechStack)
	reportContent += "---\n\n"
	reportContent += result.Document

	if err := os.WriteFile(task.OutputPath, []byte(reportContent), 0644); err != nil {
		s.failTask(task, fmt.Sprintf("写入分析结果失败: %v", err))
		return
	}

	klog.V(6).Infof("AI分析报告已保存: taskID=%s, path=%s", task.TaskID, task.OutputPath)

	task.Progress = 100
	s.completeTask(task)
}

// failTask 标记任务失败
func (s *AIAnalyzeService) failTask(task *model.AIAnalysisTask, errMsg string) {
	klog.Errorf("AI分析任务失败: taskID=%s, error=%s", task.TaskID, errMsg)

	now := time.Now()
	task.Status = "failed"
	task.ErrorMsg = errMsg
	task.CompletedAt = &now

	if err := s.taskRepo.Update(task); err != nil {
		klog.Errorf("更新任务失败状态失败: taskID=%s, error=%v", task.TaskID, err)
	}
}

// completeTask 标记任务完成
func (s *AIAnalyzeService) completeTask(task *model.AIAnalysisTask) {
	klog.V(6).Infof("AI分析任务完成: taskID=%s", task.TaskID)

	now := time.Now()
	task.Status = "completed"
	task.Progress = 100
	task.CompletedAt = &now

	if err := s.taskRepo.Update(task); err != nil {
		klog.Errorf("更新任务完成状态失败: taskID=%s, error=%v", task.TaskID, err)
	}
}

// GetAnalysisResult 获取分析结果文件内容
func (s *AIAnalyzeService) GetAnalysisResult(repoID uint) (string, error) {
	task, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return "", fmt.Errorf("获取分析任务失败: %w", err)
	}

	if task.Status != "completed" {
		return "", fmt.Errorf("分析尚未完成: status=%s", task.Status)
	}

	content, err := os.ReadFile(task.OutputPath)
	if err != nil {
		return "", fmt.Errorf("读取分析结果失败: %w", err)
	}

	return string(content), nil
}
