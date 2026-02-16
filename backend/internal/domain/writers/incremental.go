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
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/git"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

type incrementalWriter struct {
	factory      *adkagents.AgentFactory
	taskRepo     repository.TaskRepository
	repoRepo     repository.RepoRepository
	taskHintRepo repository.HintRepository
	taskService  *service.TaskService
}

func NewIncrementalWriter(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, taskHintRepo repository.HintRepository) (*incrementalWriter, error) {
	klog.V(6).Infof("[incrementalWriter.New] 开始创建增量更新服务")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[incrementalWriter.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &incrementalWriter{
		factory:      factory,
		taskRepo:     taskRepo,
		repoRepo:     repoRepo,
		taskHintRepo: taskHintRepo,
	}, nil
}

func (s *incrementalWriter) Name() domain.WriterName {
	return domain.IncrementalWriter
}

func (s *incrementalWriter) SetTaskService(taskService *service.TaskService) {
	s.taskService = taskService
}

func (s *incrementalWriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrTaskNotFound, err)
	}
	repo, err := s.repoRepo.Get(task.RepositoryID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrRepoNotFound, err)
	}

	result, err := s.createIncrementalPlan(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	for _, dir := range result.Dirs {
		task, err := s.taskService.CreateDocWriteTask(ctx, repo.ID, dir.Title, dir.Outline, dir.SortOrder)
		if err != nil {
			klog.Errorf("[%s] 创建任务失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
			continue
		}

		if err := s.saveHint(repo.ID, task, dir); err != nil {
			klog.Errorf("[%s] 保存任务提示信息失败: repoID=%d, taskID=%d, error=%v", s.Name(), repo.ID, task.ID, err)
		}
	}

	if err := s.saveAnalysisSummaryHint(repo.ID, result.AnalysisSummary); err != nil {
		klog.Errorf("[%s] 保存增量分析总结提示信息失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
	}

	return "", nil
}

func (s *incrementalWriter) createIncrementalPlan(ctx context.Context, repo *model.Repository) (*domain.DirMakerGenerationResult, error) {
	if repo.LocalPath == "" {
		return nil, fmt.Errorf("%w: repo.LocalPath 为空", domain.ErrInvalidLocalPath)
	}
	if repo.CloneCommit == "" {
		return nil, fmt.Errorf("仓库基线提交为空")
	}
	if _, err := os.Stat(repo.LocalPath); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidLocalPath, err)
	}

	klog.V(6).Infof("[%s] 执行 git pull: repoID=%d, localPath=%s", s.Name(), repo.ID, repo.LocalPath)
	pullOutput, err := git.Pull(ctx, repo.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("git pull 失败: %w", err)
	}
	klog.V(6).Infof("[%s] git pull 完成: repoID=%d, 输出=%s", s.Name(), repo.ID, pullOutput)

	if err := git.EnsureBaseCommitAvailable(ctx, repo.LocalPath, repo.CloneCommit); err != nil {
		return nil, err
	}

	summary, err := git.FormatIncrementalChangesForAI(repo.LocalPath, repo.CloneCommit)
	if err != nil {
		return nil, fmt.Errorf("增量变更分析失败: %w", err)
	}
	klog.V(6).Infof("[%s] 增量变更摘要: %s", s.Name(), summary)

	result, err := s.genIncrementalPlan(ctx, repo.LocalPath, repo.CloneCommit, summary)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[%s] 增量分析完成，生成任务数: %d", s.Name(), len(result.Dirs))
	klog.V(6).Infof("[%s] 增量分析总结: %s", s.Name(), result.AnalysisSummary)
	return result, nil
}

func (s *incrementalWriter) genIncrementalPlan(ctx context.Context, localPath string, baseCommit string, summary string) (*domain.DirMakerGenerationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "base_commit", baseCommit)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"incremental_analysis_sequential_agent",
		"增量分析顺序执行 Agent - 先生成任务，再校验修正",
		domain.AgentIncrementalEditor,
		domain.AgentIncrementalChecker,
	)
	if err != nil {
		return nil, fmt.Errorf("创建顺序 Agent 失败: %w", err)
	}

	initialMessage := fmt.Sprintf(`请基于以下仓库增量变更摘要，生成需要补充或更新的文档任务列表。

仓库路径: %s
基线提交: %s
增量摘要:
%s

请按以下步骤执行：
1. 根据增量变更摘要识别需要更新的模块与文档主题
2. 为每个主题输出文档任务 title 与 outline
3. 在 hint 中标注与变更相关的证据（文件路径、变更类型、行号范围）
4. 输出 analysis_summary 总结增量分析结论

请确保最终输出为严格符合 YAML 规范的目录结构（包含 dirs 与 analysis_summary），无多余注释或解释性文字。`, localPath, baseCommit, summary)

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
		klog.Errorf("[%s] 解析增量任务结果失败: %v", s.Name(), err)
		return nil, err
	}

	return result, nil
}

func (s *incrementalWriter) saveAnalysisSummaryHint(repoID uint, summary string) error {
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
		Title:        "增量分析总结",
		Aspect:       "增量分析总结",
		Source:       "增量分析",
		Detail:       summary,
	})
	if err := s.taskHintRepo.CreateBatch(hints); err != nil {
		klog.V(6).Infof("[%s] 保存增量分析总结失败: repoID=%d, error=%v", s.Name(), repoID, err)
		return fmt.Errorf("保存增量分析总结失败: %w", err)
	}
	return nil
}

func (s *incrementalWriter) saveHint(repoID uint, task *model.Task, spec *domain.DirMakerDirSpec) error {
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
		klog.V(6).Infof("[%s] 保存增量任务提示失败: taskID=%d, error=%v", s.Name(), task.ID, err)
		return fmt.Errorf("保存增量任务提示失败: %w", err)
	}
	return nil
}
