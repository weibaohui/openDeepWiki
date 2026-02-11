package writers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// 基于 Eino ADK 实现，用于分析代码并生成技术文档。
type defaultWriter struct {
	factory  *adkagents.AgentFactory
	hintRepo repository.HintRepository
}

func NewDefaultWriter(cfg *config.Config, hintRepo repository.HintRepository) (*defaultWriter, error) {
	klog.V(6).Infof("[dgen.New] creating document generator service")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[dgen.New] create AgentFactory failed: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}

	return &defaultWriter{
		factory:  factory,
		hintRepo: hintRepo,
	}, nil
}

func (s *defaultWriter) Name() domain.WriterName {
	return domain.DefaultWriter
}
func (s *defaultWriter) Generate(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
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
	klog.V(6).Infof("[%s] 文档生成完成: Markdown内容预览=%s", s.Name(), markdown)

	return markdown, nil
}

func (s *defaultWriter) genDocument(ctx context.Context, localPath string, title string, taskID uint) (string, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)
	adk.AddSessionValue(ctx, "task_id", taskID)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"document_generator_sequential_agent",
		"document generator sequential agent - analyze code and generate documentation",
		domain.AgentGen,
		domain.AgentCheck,
		domain.AgentDocCheck,
	)

	if err != nil {
		return "", fmt.Errorf("create agent failed: %w", err)
	}

	hintPrompt := s.buildHintPrompt(taskID)

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成一份技术文档。

仓库地址: %s
文档标题: %s
%s

请按以下步骤执行：
1. 分析仓库代码，关注可能与标题所示含义相关的内容
2. 编写详细的技术文档，使用 Markdown 格式

`, localPath, title, hintPrompt)

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

	klog.V(8).Infof("[dgen.genDoc] Agent 输出内容: \n%s\n", lastContent)
	return lastContent, nil
}

func (s *defaultWriter) buildHintPrompt(taskID uint) string {
	if s.hintRepo == nil || taskID == 0 {
		return ""
	}
	hints, err := s.hintRepo.GetByTaskID(taskID)
	if err != nil {
		klog.V(6).Infof("[dgen.buildHintPrompt] 读取任务证据失败: taskID=%d, error=%v", taskID, err)
		return ""
	}
	if len(hints) == 0 {
		return ""
	}
	builder := &strings.Builder{}
	builder.WriteString("撰写文章时可参考如下线索: \n")
	for _, ev := range hints {
		builder.WriteString("- 维度: ")
		builder.WriteString(safe(ev.Aspect))
		builder.WriteString("\n  来源: ")
		builder.WriteString(safe(ev.Source))
		builder.WriteString("\n  细节: ")
		builder.WriteString(safe(ev.Detail))
		builder.WriteString("\n")
	}
	return builder.String()
}
