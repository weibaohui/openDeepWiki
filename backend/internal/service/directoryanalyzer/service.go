package directoryanalyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

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
	agentTaskGenerator = "TaskGenerator" // 任务生成 Agent
	agentTaskValidator = "TaskValidator" // 任务校验 Agent
)

// 错误定义
var (
	ErrInvalidLocalPath     = errors.New("无效的本地路径")
	ErrAgentExecutionFailed = errors.New("Agent 执行失败")
	ErrJSONParseFailed      = errors.New("JSON 解析失败")
	ErrTaskCreationFailed   = errors.New("任务创建失败")
	ErrNoAgentOutput        = errors.New("Agent 未产生任何输出内容")
)

// Service 目录分析任务生成服务。
// 基于 Eino ADK 实现，用于分析代码目录并动态生成分析任务。
type Service struct {
	factory  *adkagents.AgentFactory
	taskRepo repository.TaskRepository
}

// New 创建目录分析服务实例。
func New(cfg *config.Config, taskRepo repository.TaskRepository) (*Service, error) {
	klog.V(6).Infof("[directoryanalyzer.New] 开始创建目录分析服务")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[directoryanalyzer.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &Service{
		factory:  factory,
		taskRepo: taskRepo,
	}, nil
}

// CreateTasks 分析仓库目录并创建任务。
func (s *Service) CreateTasks(ctx context.Context, repo *model.Repository) ([]*model.Task, error) {
	if repo == nil {
		return nil, fmt.Errorf("%w: repo 为空", ErrInvalidLocalPath)
	}
	if repo.LocalPath == "" {
		return nil, ErrInvalidLocalPath
	}
	if _, err := os.Stat(repo.LocalPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLocalPath, err)
	}

	result, err := s.generateTaskPlan(ctx, repo.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[directoryanalyzer.CreateTasks] 分析完成，生成任务数: %d", len(result.Tasks))
	klog.V(6).Infof("[directoryanalyzer.CreateTasks] 分析摘要: %s", result.AnalysisSummary)

	createdTasks := make([]*model.Task, 0, len(result.Tasks))
	var creationErrors []error

	for _, spec := range result.Tasks {
		task := &model.Task{
			RepositoryID: repo.ID,
			Type:         spec.Type,
			Title:        spec.Title,
			Status:       string(statemachine.TaskStatusPending),
			SortOrder:    spec.SortOrder,
		}

		if err := s.taskRepo.Create(task); err != nil {
			klog.Errorf("[directoryanalyzer.CreateTasks] 创建任务失败: type=%s, error=%v", spec.Type, err)
			creationErrors = append(creationErrors, fmt.Errorf("创建任务 %s 失败: %w", spec.Type, err))
			continue
		}

		klog.V(6).Infof("[directoryanalyzer.CreateTasks] 任务创建成功: id=%d, type=%s, title=%s", task.ID, task.Type, task.Title)
		createdTasks = append(createdTasks, task)
	}

	if len(creationErrors) > 0 {
		klog.Warningf("[directoryanalyzer.CreateTasks] 部分任务创建失败: 成功=%d, 失败=%d", len(createdTasks), len(creationErrors))
		if len(createdTasks) == 0 {
			return nil, fmt.Errorf("%w: %w", ErrTaskCreationFailed, creationErrors[0])
		}
	}

	klog.V(6).Infof("[directoryanalyzer.CreateTasks] 全部完成，成功创建任务数: %d", len(createdTasks))
	return createdTasks, nil
}

// generateTaskPlan 执行任务生成链路，返回解析后的任务列表结果。
func (s *Service) generateTaskPlan(ctx context.Context, localPath string) (*generationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"TaskGeneratorSequentialAgent",
		"任务生成顺序执行 Agent - 先生成任务列表，再校验修正",
		agentTaskGenerator,
		agentTaskValidator,
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
				result, parseErr := parseTaskPlan(lastContent)
				if parseErr == nil {
					return result, nil
				}
			}
		}
		return nil, fmt.Errorf("Agent 执行出错: %w", err)
	}

	if lastContent == "" {
		return nil, ErrNoAgentOutput
	}

	result, err := parseTaskPlan(lastContent)
	if err != nil {
		klog.Errorf("[directoryanalyzer.generateTaskPlan] 解析任务生成结果失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[directoryanalyzer.generateTaskPlan] 执行成功，生成任务数: %d", len(result.Tasks))
	return result, nil
}

// parseTaskPlan 从 Agent 输出解析任务生成结果。
func parseTaskPlan(content string) (*generationResult, error) {
	klog.V(6).Infof("[directoryanalyzer.parseTaskPlan] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 尝试从内容中提取 JSON
	jsonStr := utils.ExtractJSON(content)
	if jsonStr == "" {
		klog.Warningf("[directoryanalyzer.parseTaskPlan] 未能从内容中提取 JSON")
		return nil, fmt.Errorf("%w: 未能从 Agent 输出中提取有效 JSON", ErrJSONParseFailed)
	}

	var result generationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		klog.Errorf("[directoryanalyzer.parseTaskPlan] JSON 解析失败: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrJSONParseFailed, err)
	}

	klog.V(6).Infof("[directoryanalyzer.parseTaskPlan] 解析成功，任务数: %d", len(result.Tasks))
	return &result, nil
}
