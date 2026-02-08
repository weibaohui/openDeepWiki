package apianalyzer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

const (
	agentExplorer = "api_explorer"
	agentDocCheck = "document_checker"
	agentMdCheck  = "markdown_checker"
)

var (
	ErrInvalidLocalPath     = errors.New("invalid local path")
	ErrAgentExecutionFailed = errors.New("agent execution failed")
	ErrNoAgentOutput        = errors.New("no agent output")
)

type Service struct {
	factory  *adkagents.AgentFactory
	hintRepo repository.HintRepository
}

// New 创建API接口分析服务实例。
func New(cfg *config.Config, hintRepo repository.HintRepository) (*Service, error) {
	klog.V(6).Infof("[apianalyzer.New] 创建API接口分析服务")
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[apianalyzer.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}
	return &Service{
		factory:  factory,
		hintRepo: hintRepo,
	}, nil
}

// Generate 生成API接口分析文档。
func (s *Service) Generate(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", ErrInvalidLocalPath)
	}
	if title == "" {
		return "", fmt.Errorf("%w: title is empty", ErrInvalidLocalPath)
	}
	if repoID == 0 {
		return "", fmt.Errorf("%w: repo id is empty", ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[apianalyzer.Generate] 开始生成文档: 仓库路径=%s, 标题=%s, 任务ID=%d", localPath, title, taskID)
	markdown, err := s.genDocument(ctx, localPath, title, repoID, taskID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[apianalyzer.Generate] 文档生成完成: 内容长度=%d", len(markdown))
	return markdown, nil
}

// genDocument 负责调用Agent并返回最终文档内容。
func (s *Service) genDocument(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"api_parser_sequential_agent",
		"api parser sequential agent - explore http APIs then validate document and markdown",
		agentExplorer,
		agentDocCheck,
		agentMdCheck,
	)
	if err != nil {
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	hintYAML := s.buildHintYAML(repoID)
	var hintSection string
	if hintYAML == "" {
		hintSection = ""
	} else {
		hintSection = fmt.Sprintf("线索信息（YAML）:\n```yaml\n%s```\n", hintYAML)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成 API 接口说明文档。

仓库地址: %s
文档标题: %s
%s
请按以下要求输出：
1. 按模块划分，每个模块使用一个表格
2. 每行包含：名称、访问路径、功能、参数列举、说明
3. 明确标注对外可访问接口（例如 HTTP 路由）
4. 未发现的模块或接口需明确说明
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
		return "", ErrNoAgentOutput
	}

	klog.V(8).Infof("[apianalyzer.genDocument] Agent 输出内容: \n%s\n", lastContent)
	return lastContent, nil
}

// buildHintYAML 构建API接口分析线索的YAML输入。
func (s *Service) buildHintYAML(repoID uint) string {
	if s.hintRepo == nil || repoID == 0 {
		return ""
	}
	keywords := apiKeywords()
	hints, err := s.hintRepo.SearchInRepo(repoID, keywords)
	if err != nil {
		klog.V(6).Infof("[apianalyzer.buildHintYAML] 仓库范围搜索证据失败: repoID=%d, error=%v", repoID, err)
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
		klog.V(6).Infof("[apianalyzer.buildHintYAML] 生成 YAML 失败: error=%v", err)
		return ""
	}
	return string(data)
}

// apiKeywords 返回API接口分析使用的关键词集合。
func apiKeywords() []string {
	return []string{
		"api",
		"route",
		"router",
		"handler",
		"controller",
		"http",
		"endpoint",
		"gin",
		"接口",
		"路由",
		"GET",
		"POST",
		"PUT",
		"DELETE",
	}
}

// safe 统一将空字段转换为占位描述。
func safe(value string) string {
	if value == "" {
		return "(无)"
	}
	return value
}
