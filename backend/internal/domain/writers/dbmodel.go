package writers

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

type dbModelWriter struct {
	factory  *adkagents.AgentFactory
	hintRepo repository.HintRepository
	taskRepo repository.TaskRepository
}

func NewDBModelWriter(cfg *config.Config, hintRepo repository.HintRepository, taskRepo repository.TaskRepository) (*dbModelWriter, error) {
	klog.V(6).Infof("[NewDBModelWriter] 创建数据库模型解析服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[NewDBModelWriter] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &dbModelWriter{
		factory:  factory,
		hintRepo: hintRepo,
		taskRepo: taskRepo,
	}, nil
}

func (s *dbModelWriter) Name() domain.WriterName {
	return domain.DBModelWriter
}

func (s *dbModelWriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", domain.ErrInvalidLocalPath)
	}
	if title == "" {
		return "", fmt.Errorf("%w: title is empty", domain.ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[%s] 开始生成文档: 仓库路径=%s, 标题=%s, 任务ID=%d", s.Name(), localPath, title, taskID)
	markdown, err := s.genDocument(ctx, localPath, title, taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[%s] 文档生成完成: 内容长度=%d", s.Name(), len(markdown))
	return markdown, nil
}

func (s *dbModelWriter) genDocument(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"db_model_parser_sequential_agent",
		"db model parser sequential agent - explore database models then validate markdown",
		domain.AgentDBModelExplorer,
		domain.AgentDocCheck,
		domain.AgentMdCheck,
	)
	if err != nil {
		return "", fmt.Errorf("[%s] create agent failed: %w", s.Name(), err)
	}

	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("[%s] get task by id failed: %w", s.Name(), err)
	}
	hintYAML := s.buildHintYAML(task.RepositoryID)
	var hintSection string
	if hintYAML == "" {
		hintSection = ""
	} else {
		hintSection = fmt.Sprintf("线索信息（YAML）:\n```yaml\n%s```\n", hintYAML)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成数据库表结构说明文档。

仓库地址: %s
文档标题: %s
%s
请按以下步骤执行：
1. 根据线索与源码，定位数据库表定义或模型定义
2. 输出完整的 Markdown 文档，包含所有表结构与字段说明
`, localPath, title, hintSection)

	lastContent, err := adkagents.RunAgentToLastContent(ctx, agent, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})
	if err != nil {
		return "", fmt.Errorf("agent execution error: %w", err)
	}
	if lastContent == "" {
		return "", domain.ErrNoAgentOutput
	}

	klog.V(8).Infof("[dbmodel.genDocument] Agent 输出内容: \n%s\n", lastContent)
	return lastContent, nil
}

func (s *dbModelWriter) buildHintYAML(repoID uint) string {
	if s.hintRepo == nil || repoID == 0 {
		return ""
	}
	keywords := dbKeywords()
	hints, err := s.hintRepo.SearchInRepo(repoID, keywords)
	if err != nil {
		klog.V(6).Infof("[dbmodel.buildHintYAML] 仓库范围搜索证据失败: repoID=%d, error=%v", repoID, err)
		return ""
	}
	if len(hints) == 0 {
		return ""
	}

	items := make([]map[string]string, 0, len(hints))
	for _, ev := range hints {
		items = append(items, map[string]string{
			"title":  safe(ev.Title),
			"detail": safe(ev.Detail),
			"source": safe(ev.Source),
		})
	}

	if len(items) == 0 {
		return ""
	}

	payload := map[string][]map[string]string{
		"hints": items,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		klog.V(6).Infof("[dbmodel.buildHintYAML] 生成 YAML 失败: error=%v", err)
		return ""
	}
	return string(data)
}

func dbKeywords() []string {
	return []string{
		"sql",
		"ddl",
		"schema",
		"model",
		"migration",
		"table",
		"column",
		"database",
		"数据库",
		"表",
		"字段",
		"数据模型",
	}
}
