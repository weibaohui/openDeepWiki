package dbmodelparser

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

const (
	agentExplorer = "db_model_explorer"
	agentCheck    = "markdown_checker"
)

var (
	ErrInvalidLocalPath     = errors.New("invalid local path")
	ErrAgentExecutionFailed = errors.New("agent execution failed")
	ErrNoAgentOutput        = errors.New("no agent output")
)

type Service struct {
	factory      *adkagents.AgentFactory
	evidenceRepo repository.EvidenceRepository
}

func New(cfg *config.Config, evidenceRepo repository.EvidenceRepository) (*Service, error) {
	klog.V(6).Infof("[dbmodel.New] 创建数据库模型解析服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[dbmodel.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &Service{
		factory:      factory,
		evidenceRepo: evidenceRepo,
	}, nil
}

func (s *Service) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", ErrInvalidLocalPath)
	}
	if title == "" {
		return "", fmt.Errorf("%w: title is empty", ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[dbmodel.Generate] 开始生成文档: 仓库路径=%s, 标题=%s, 任务ID=%d", localPath, title, taskID)
	markdown, err := s.genDocument(ctx, localPath, title, taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[dbmodel.Generate] 文档生成完成: 内容长度=%d", len(markdown))
	return markdown, nil
}

func (s *Service) genDocument(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"db_model_parser_sequential_agent",
		"db model parser sequential agent - explore database models then validate markdown",
		agentExplorer,
		agentCheck,
	)
	if err != nil {
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	evidenceYAML := s.buildEvidenceYAML(taskID)
	var evidenceSection string
	if evidenceYAML == "" {
		evidenceSection = "线索信息: 未找到数据库相关证据，请自行从源码中探索表结构定义。\n"
	} else {
		evidenceSection = fmt.Sprintf("线索信息（YAML）:\n```yaml\n%s```\n", evidenceYAML)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成数据库表结构说明文档。

仓库地址: %s
文档标题: %s
%s
请按以下步骤执行：
1. 根据线索与源码，定位数据库表定义或模型定义
2. 输出完整的 Markdown 文档，包含所有表结构与字段说明
`, localPath, title, evidenceSection)

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
		return "", ErrNoAgentOutput
	}

	klog.V(8).Infof("[dbmodel.genDocument] Agent 输出内容: \n%s\n", lastContent)
	return lastContent, nil
}

func (s *Service) buildEvidenceYAML(taskID uint) string {
	if s.evidenceRepo == nil || taskID == 0 {
		return ""
	}
	evidences, err := s.evidenceRepo.GetByTaskID(taskID)
	if err != nil {
		klog.V(6).Infof("[dbmodel.buildEvidenceYAML] 读取任务证据失败: taskID=%d, error=%v", taskID, err)
		return ""
	}
	if len(evidences) == 0 {
		return ""
	}

	items := make([]map[string]string, 0, len(evidences))
	for _, ev := range evidences {
		if !s.matchDatabaseEvidence(ev) {
			continue
		}
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
		"evidences": items,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		klog.V(6).Infof("[dbmodel.buildEvidenceYAML] 生成 YAML 失败: error=%v", err)
		return ""
	}
	return string(data)
}

func (s *Service) matchDatabaseEvidence(ev model.TaskEvidence) bool {
	combined := strings.ToLower(strings.Join([]string{ev.Title, ev.Aspect, ev.Source, ev.Detail}, " "))
	keywords := []string{
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
	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}
	return false
}

func safe(value string) string {
	if value == "" {
		return "(无)"
	}
	return value
}
