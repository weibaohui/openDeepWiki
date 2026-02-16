package writers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/git"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type incrementalWriter struct {
	factory      *adkagents.AgentFactory
	taskRepo     repository.TaskRepository
	repoRepo     repository.RepoRepository
	docRepo      repository.DocumentRepository
	taskHintRepo repository.HintRepository
	taskService  *service.TaskService
}

func NewIncrementalWriter(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, taskHintRepo repository.HintRepository, docRepo repository.DocumentRepository) (*incrementalWriter, error) {
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
		docRepo:      docRepo,
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

	for _, dir := range result.AddDirs {
		task, err := s.taskService.CreateDocWriteTask(ctx, repo.ID, dir.Title, dir.Outline, dir.SortOrder)
		if err != nil {
			klog.Errorf("[%s] 创建任务失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
			continue
		}

		if err := s.saveHint(repo.ID, task, dir); err != nil {
			klog.Errorf("[%s] 保存任务提示信息失败: repoID=%d, taskID=%d, error=%v", s.Name(), repo.ID, task.ID, err)
		}
	}

	for _, dir := range result.UpdateDirs {

		// 写入数据库Task表，增加一种改写类型，类似TitleRewriter，ContentRewriter
		klog.V(6).Infof("待更新目录:[%d] %s\n", dir.DocID, dir.Title)
		klog.V(6).Infof("待更新内容: %s\n", dir.Content)
		klog.V(6).Infof("待更新目标内容: %s\n", dir.Replace)

		_, err = s.taskService.CreateContentRewriteTask(ctx, repo.ID, dir.Title, dir.Content, dir.Replace, dir.DocID)
		if err != nil {
			klog.Errorf("[%s] 创建任务失败: repoID=%d, error=%v", s.Name(), repo.ID, err)
			continue
		}

	}

	return "", nil
}

func (s *incrementalWriter) createIncrementalPlan(ctx context.Context, repo *model.Repository) (*domain.IncrementalGenerationResult, error) {
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

	docList, err := s.docRepo.GetAllDocumentsTitleAndID(repo.ID)
	if err != nil {
		return nil, fmt.Errorf("获取所有文档标题与ID失败: %w", err)
	}
	//将文档标题、ID拼接为字符串，每个文档占用一行，格式为：title: id
	var docListStr strings.Builder
	for _, doc := range docList {
		fmt.Fprintf(&docListStr, "- 标题=%s\t ID=%d\n", doc.Title, doc.ID)
	}

	result, err := s.genIncrementalPlan(ctx, repo.LocalPath, repo.CloneCommit, summary, docListStr.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	return result, nil
}

// genIncrementalPlan 生成增量更新计划。
// summary 增量变更摘要
// docList 当前仓库的所有文档的标题与ID列表
func (s *incrementalWriter) genIncrementalPlan(ctx context.Context, localPath string, baseCommit string, summary string, docList string) (*domain.IncrementalGenerationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "incremental_summary", summary)

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
当前已生成文档列表：
%s
增量摘要:
%s

请按以下步骤执行：
1. 根据增量变更摘要识别需要更新的模块与文档主题
2. 为每个主题输出文档任务 title 与 outline
3. 在 hint 中标注与变更相关的证据（文件路径、变更类型、行号范围）
4. 输出 analysis_summary 总结增量分析结论

请确保最终输出为严格符合 YAML 规范的目录结构（包含 dirs 与 analysis_summary），无多余注释或解释性文字。`, localPath, baseCommit, docList, summary)

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
	result, err := parseIncrementalDirList(lastContent)
	if err != nil {
		klog.Errorf("[%s] 解析增量任务结果失败: %v", s.Name(), err)
		return nil, err
	}

	return result, nil
}

// parseIncrementalDirList 从 Agent 输出解析增量目录生成结果。
func parseIncrementalDirList(content string) (*domain.IncrementalGenerationResult, error) {
	klog.V(6).Infof("[dm.parseIncrementalDirList] 开始解析 Agent 输出内容，长度: %d", len(content))

	// 尝试从内容中提取 YAML
	yamlStr := utils.ExtractYAML(content)
	if yamlStr == "" {
		klog.Warningf("[dm.parseIncrementalDirList] 提取 YAML 失败")
		return nil, fmt.Errorf("%w: 提取 YAML 失败", domain.ErrYAMLParseFailed)
	}

	var result domain.IncrementalGenerationResult
	if err := yaml.Unmarshal([]byte(yamlStr), &result); err != nil {
		klog.Errorf("[dm.parseIncrementalDirList] YAML 解析失败: %v", err)
		return nil, fmt.Errorf("%w: %v", domain.ErrYAMLParseFailed, err)
	}

	klog.V(6).Infof("[dm.parseIncrementalDirList] 解析完成，更新目录数: %d, 新增目录数: %d", len(result.UpdateDirs), len(result.AddDirs))
	return &result, nil
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
