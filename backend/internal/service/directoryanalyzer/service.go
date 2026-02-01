package directoryanalyzer

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/repository"
	"github.com/opendeepwiki/backend/internal/service/statemachine"
	"k8s.io/klog/v2"
)

// Service 错误定义
var (
	ErrInvalidLocalPath     = errors.New("无效的本地路径")
	ErrAgentExecutionFailed = errors.New("Agent 执行失败")
	ErrJSONParseFailed      = errors.New("JSON 解析失败")
	ErrTaskCreationFailed   = errors.New("任务创建失败")
	ErrWorkflowNotBuilt     = errors.New("Workflow 未构建")
)

// DirectoryAnalyzerService 目录分析任务生成服务
// 基于 Eino ADK 实现，用于分析代码目录并动态生成分析任务
type DirectoryAnalyzerService struct {
	cfg      *config.Config
	workflow *TaskGeneratorWorkflow
	taskRepo repository.TaskRepository
}

// NewDirectoryAnalyzerService 创建目录分析服务实例
// cfg: 配置信息
// taskRepo: 任务仓库接口
// 返回: DirectoryAnalyzerService 实例或错误
func NewDirectoryAnalyzerService(
	cfg *config.Config,
	taskRepo repository.TaskRepository,
) (*DirectoryAnalyzerService, error) {
	klog.V(6).Infof("[NewDirectoryAnalyzerService] 开始创建服务")

	// 创建 Workflow
	workflow, err := NewTaskGeneratorWorkflow(cfg)
	if err != nil {
		klog.Errorf("[NewDirectoryAnalyzerService] 创建 Workflow 失败: %v", err)
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	klog.V(6).Infof("[NewDirectoryAnalyzerService] 服务创建成功")

	return &DirectoryAnalyzerService{
		cfg:      cfg,
		workflow: workflow,
		taskRepo: taskRepo,
	}, nil
}

// AnalyzeAndCreateTasks 分析目录并创建任务
// ctx: 上下文
// localPath: 本地仓库路径
// repo: 仓库模型（包含 ID 等信息）
// 返回: 创建的任务列表或错误
func (s *DirectoryAnalyzerService) AnalyzeAndCreateTasks(
	ctx context.Context,
	localPath string,
	repo *model.Repository,
) ([]*model.Task, error) {
	klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 开始分析: localPath=%s, repoID=%d", localPath, repo.ID)

	// 1. 校验本地路径
	if err := s.validateLocalPath(localPath); err != nil {
		klog.Warningf("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 路径校验失败: %v", err)
		return nil, err
	}

	// 2. 执行 Workflow 分析
	result, err := s.workflow.Run(ctx, localPath)
	if err != nil {
		klog.Errorf("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] Workflow 执行失败: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 分析完成，生成任务数: %d", len(result.Tasks))
	klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 分析摘要: %s", result.AnalysisSummary)

	// 3. 遍历创建任务
	createdTasks := make([]*model.Task, 0, len(result.Tasks))
	var creationErrors []error

	for _, generatedTask := range result.Tasks {
		task := &model.Task{
			RepositoryID: repo.ID,
			Type:         generatedTask.Type,
			Title:        generatedTask.Title,
			Status:       string(statemachine.TaskStatusPending),
			SortOrder:    generatedTask.SortOrder,
		}

		if err := s.taskRepo.Create(task); err != nil {
			klog.Errorf("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 创建任务失败: type=%s, error=%v", generatedTask.Type, err)
			creationErrors = append(creationErrors, fmt.Errorf("创建任务 %s 失败: %w", generatedTask.Type, err))
			continue
		}

		klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 任务创建成功: id=%d, type=%s, title=%s", task.ID, task.Type, task.Title)
		createdTasks = append(createdTasks, task)
	}

	// 4. 处理创建结果
	if len(creationErrors) > 0 {
		klog.Warningf("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 部分任务创建失败: 成功=%d, 失败=%d", len(createdTasks), len(creationErrors))
		// 如果有部分任务创建成功，仍然返回已创建的任务列表
		if len(createdTasks) == 0 {
			return nil, fmt.Errorf("%w: %v", ErrTaskCreationFailed, creationErrors[0])
		}
	}

	klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeAndCreateTasks] 全部完成，成功创建任务数: %d", len(createdTasks))
	return createdTasks, nil
}

// validateLocalPath 校验本地路径是否有效
func (s *DirectoryAnalyzerService) validateLocalPath(localPath string) error {
	if localPath == "" {
		return fmt.Errorf("%w: 路径为空", ErrInvalidLocalPath)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: 路径不存在: %s", ErrInvalidLocalPath, localPath)
		}
		return fmt.Errorf("%w: 无法访问路径: %s, error: %v", ErrInvalidLocalPath, localPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: 路径不是目录: %s", ErrInvalidLocalPath, localPath)
	}

	return nil
}

// AnalyzeOnly 仅分析目录，不创建任务
// 用于预览或测试
func (s *DirectoryAnalyzerService) AnalyzeOnly(
	ctx context.Context,
	localPath string,
) (*TaskGenerationResult, error) {
	klog.V(6).Infof("[DirectoryAnalyzerService.AnalyzeOnly] 开始分析: localPath=%s", localPath)

	// 校验本地路径
	if err := s.validateLocalPath(localPath); err != nil {
		return nil, err
	}

	// 执行 Workflow 分析
	result, err := s.workflow.Run(ctx, localPath)
	if err != nil {
		klog.Errorf("[DirectoryAnalyzerService.AnalyzeOnly] Workflow 执行失败: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrAgentExecutionFailed, err)
	}

	return result, nil
}
