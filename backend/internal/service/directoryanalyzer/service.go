package directoryanalyzer

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
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

	// 2. 执行 Workflow 分析
	result, err := s.workflow.Run(ctx, localPath)
	if err != nil {
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
