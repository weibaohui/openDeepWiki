package writers

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// tocWriter 目录分析任务生成服务。
type tocWriter struct {
	factory      *adkagents.AgentFactory
	docRepo      repository.DocumentRepository
	taskRepo     repository.TaskRepository
	repoRepo     repository.RepoRepository
	taskHintRepo repository.HintRepository
	taskService  *service.TaskService
}

// New 创建目录分析服务实例。
func NewTocWriter(cfg *config.Config, docRepo repository.DocumentRepository, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, taskHintRepo repository.HintRepository) (*tocWriter, error) {
	klog.V(6).Infof("[tocWriter.New] 开始创建目录分析服务")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[tocWriter.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &tocWriter{
		factory:      factory,
		docRepo:      docRepo,
		repoRepo:     repoRepo,
		taskRepo:     taskRepo,
		taskHintRepo: taskHintRepo,
	}, nil
}

func (s *tocWriter) Name() domain.WriterName {
	return domain.TocWriter
}
func (s *tocWriter) SetTaskService(taskService *service.TaskService) {
	s.taskService = taskService
}
func (s *tocWriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrTaskNotFound, err)
	}
	repo, err := s.repoRepo.Get(task.RepositoryID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrRepoNotFound, err)
	}

	result, err := s.createDirs(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrDirMakerGenerationFailed, err)
	}
	for _, dir := range result.Dirs {
		task, err := s.taskService.CreateDocWriteTask(ctx, repo.ID, dir.Title, dir.SortOrder)
		if err != nil {
			klog.Errorf("[%s] 创建任务失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
			continue
		}

		if err := s.saveHint(repo.ID, task, dir); err != nil {
			klog.Errorf("[%s] 保存任务提示信息失败: repoID=%d, taskID=%d, error=%v", s.Name(), repo.ID, task.ID, err)
		}
	}
	// result.AnalysisSummary 保存到提示中
	if err := s.saveAnalysisSummaryHint(repo.ID, result.AnalysisSummary); err != nil {
		klog.Errorf("[%s] 保存目录分析总结提示信息失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
	}

	return "", nil
}

// CreateDirs 分析仓库目录并创建目录。
func (s *tocWriter) createDirs(ctx context.Context, repo *model.Repository) (*domain.DirMakerGenerationResult, error) {

	if repo.LocalPath == "" {
		return nil, fmt.Errorf("%w: repo.LocalPath 为空", domain.ErrInvalidLocalPath)
	}
	if _, err := os.Stat(repo.LocalPath); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidLocalPath, err)
	}

	result, err := s.genDirList(ctx, repo.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[%s] 分析完成，生成目录数: %d", s.Name(), len(result.Dirs))
	klog.V(6).Infof("[%s] 分析摘要: %s", s.Name(), result.AnalysisSummary)
	return result, nil
}

// generateTaskPlan 执行任务生成链路，返回解析后的任务列表结果。
func (s *tocWriter) genDirList(ctx context.Context, localPath string) (*domain.DirMakerGenerationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"toc_generator_sequential_agent",
		"目录制定者顺序执行 Agent - 先生成目录，再校验修正",
		domain.AgentTocEditor,
		domain.AgentTocChecker,
	)

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成需要的技术分析任务列表。

仓库地址: %s

请按以下步骤执行：
1. 分析仓库目录结构，识别项目类型和技术栈
2. 根据项目特征生成初步的任务列表
3. 校验并修正任务列表，确保完整性和合理性

请确保最终输出为严格符合 YAML 规范的目录结构（包含 dirs 与 analysis_summary），无多余注释或解释性文字。`, localPath)
	lastContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Agent 执行出错: %w", err)
	}

	if lastContent == "" {
		return nil, domain.ErrNoAgentOutput
	}
	klog.V(6).Infof("[%s] 最后 Agent 输出 原文: \n%s\n", s.Name(), lastContent)

	result, err := parseDirList(lastContent)
	if err != nil {
		klog.Errorf("[%s] 解析 目录生成 结果失败: %v", s.Name(), err)
		return nil, err
	}

	klog.V(6).Infof("[%s] 执行成功，生成目录数: %d", s.Name(), len(result.Dirs))
	return result, nil
}

// parseDirList 从 Agent 输出解析目录生成结果。
func parseDirList(content string) (*domain.DirMakerGenerationResult, error) {
	klog.V(6).Infof("[dm.parseList] 开始解析 Agent 输出内容，长度: %d", len(content))

	// 尝试从内容中提取 YAML
	yamlStr := utils.ExtractYAML(content)
	if yamlStr == "" {
		klog.Warningf("[dm.parseList] 提取 YAML 失败")
		return nil, fmt.Errorf("%w: 提取 YAML 失败", domain.ErrYAMLParseFailed)
	}

	var result domain.DirMakerGenerationResult
	if err := yaml.Unmarshal([]byte(yamlStr), &result); err != nil {
		klog.Errorf("[dm.parseList] YAML 解析失败: %v", err)
		return nil, fmt.Errorf("%w: %v", domain.ErrYAMLParseFailed, err)
	}
	klog.V(6).Infof("AI分析概要\n%s\n", result.AnalysisSummary)
	klog.V(6).Infof("[dm.parseList] 解析完成，目录数: %d", len(result.Dirs))
	return &result, nil
}

func (s *tocWriter) saveAnalysisSummaryHint(repoID uint, summary string) error {
	if s.taskHintRepo == nil {
		return nil
	}
	if summary == "" {
		return nil
	}
	hints := make([]model.TaskHint, 0, 1)
	hints = append(hints, model.TaskHint{
		RepositoryID: repoID,
		TaskID:       0,
		Title:        "目录分析总结",
		Aspect:       "目录分析总结",
		Source:       "目录分析",
		Detail:       summary,
	})
	if err := s.taskHintRepo.CreateBatch(hints); err != nil {
		klog.V(6).Infof("[%s] 保存 任务总结 信息失败: repoID=%d, error=%v", s.Name(), repoID, err)
		return fmt.Errorf("保存 任务总结 信息失败: %w", err)
	}
	return nil
}

func (s *tocWriter) saveHint(repoID uint, task *model.Task, spec *domain.DirMakerDirSpec) error {
	if s.taskHintRepo == nil {
		return nil
	}
	if len(spec.Hint) == 0 {
		return nil
	}
	hints := make([]model.TaskHint, 0, len(spec.Hint))
	for _, item := range spec.Hint {
		hints = append(hints, model.TaskHint{
			RepositoryID: repoID,
			TaskID:       task.ID,
			Title:        spec.Title,
			Aspect:       item.Aspect,
			Source:       item.Source,
			Detail:       item.Detail,
		})
	}
	if err := s.taskHintRepo.CreateBatch(hints); err != nil {
		klog.V(6).Infof("[%s] 保存 任务提示 信息失败: taskID=%d, error=%v", s.Name(), task.ID, err)
		return fmt.Errorf("保存 任务提示 信息失败: %w", err)
	}
	return nil
}
