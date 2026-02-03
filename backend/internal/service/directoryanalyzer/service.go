package directoryanalyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// Agent 名称常量
const (
	AgentTaskGenerator = "TaskGenerator" // 任务生成 Agent
	AgentTaskValidator = "TaskValidator" // 任务校验 Agent
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
	factory  *adkagents.AgentFactory
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

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[NewDirectoryAnalyzerService] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("failed to create agent factory: %w", err)
	}

	return &DirectoryAnalyzerService{
		cfg:      cfg,
		factory:  factory,
		taskRepo: taskRepo,
	}, nil
}

// runTaskGeneration 执行任务生成链路，返回解析后的任务列表结果
// ctx: 上下文
// localPath: 仓库本地路径
// 返回: 任务生成结果或错误
func (s *DirectoryAnalyzerService) runTaskGeneration(ctx context.Context, localPath string) (*TaskGenerationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"TaskGeneratorSequentialAgent",
		"任务生成顺序执行 Agent - 先生成任务列表，再校验修正",
		AgentTaskGenerator,
		AgentTaskValidator,
	)

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成需要的技术分析任务列表。

仓库地址: %s

请按以下步骤执行：
1. 分析仓库目录结构，识别项目类型和技术栈
2. 根据项目特征生成初步的任务列表
3. 校验并修正任务列表，确保完整性和合理性

请确保最终输出格式为有效的 JSON。`, localPath)
	lastContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})
	if err != nil {
		if adkagents.IsMaxIterationsError(err) {
			if lastContent != "" {
				result, parseErr := ParseTaskGenerationResult(lastContent)
				if parseErr == nil {
					return result, nil
				}
			}
		}
		return nil, fmt.Errorf("Agent 执行出错: %w", err)
	}

	if lastContent == "" {
		return nil, fmt.Errorf("Agent 未产生任何输出内容")
	}

	result, err := ParseTaskGenerationResult(lastContent)
	if err != nil {
		klog.Errorf("[DirectoryAnalyzerService.runTaskGeneration] 解析任务生成结果失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[DirectoryAnalyzerService.runTaskGeneration] 执行成功，生成任务数: %d", len(result.Tasks))
	return result, nil
}

// AnalyzeAndCreateTasks 分析目录并创建任务
// ctx: 上下文
// localPath: 本地仓库路径
// repo: 仓库模型（包含 ID 等信息）
// 返回: 创建的任务列表或错误
func (s *DirectoryAnalyzerService) AnalyzeAndCreateTasks(ctx context.Context, localPath string, repo *model.Repository) ([]*model.Task, error) {
	result, err := s.runTaskGeneration(ctx, localPath)
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

// ParseTaskGenerationResult 从 Agent 输出解析任务生成结果
// content: Agent 返回的原始内容
// 返回: 解析后的结果或错误
func ParseTaskGenerationResult(content string) (*TaskGenerationResult, error) {
	klog.V(6).Infof("[ParseTaskGenerationResult] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 尝试从内容中提取 JSON
	jsonStr := utils.ExtractJSON(content)
	if jsonStr == "" {
		klog.Warningf("[ParseTaskGenerationResult] 未能从内容中提取 JSON")
		return nil, fmt.Errorf("未能从 Agent 输出中提取有效 JSON")
	}

	var result TaskGenerationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		klog.Errorf("[ParseTaskGenerationResult] JSON 解析失败: %v", err)
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	klog.V(6).Infof("[ParseTaskGenerationResult] 解析成功，任务数: %d", len(result.Tasks))
	return &result, nil
}
