package documentgenerator

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// Agent 名称常量
const (
	agentDocumentGenerator = "document_generator" // 文档生成 Agent
)

// 错误定义
var (
	ErrInvalidLocalPath     = errors.New("无效的本地路径")
	ErrAgentExecutionFailed = errors.New("Agent 执行失败")
	ErrEmptyDocumentContent  = errors.New("文档内容为空")
	ErrNoAgentOutput        = errors.New("Agent 未产生任何输出内容")
)

// generationResult 表示 Agent 输出的文档生成结果（仅包内使用）。
type generationResult struct {
	Content         string `json:"content"`          // 生成的文档内容
	AnalysisSummary string `json:"analysis_summary"` // 分析摘要
}

// Service 文档生成服务。
// 基于 Eino ADK 实现，用于分析代码并生成技术文档。
type Service struct {
	factory *adkagents.AgentFactory
}

// New 创建文档生成服务实例。
func New(cfg *config.Config) (*Service, error) {
	klog.V(6).Infof("[documentgenerator.New] 开始创建文档生成服务")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[documentgenerator.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &Service{
		factory: factory,
	}, nil
}

// Generate 分析仓库代码并生成文档。
func (s *Service) Generate(ctx context.Context, localPath string, title string) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: localPath 为空", ErrInvalidLocalPath)
	}
	if title == "" {
		return "", fmt.Errorf("%w: title 为空", ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[documentgenerator.Generate] 开始生成文档: localPath=%s, title=%s", localPath, title)

	result, err := s.genDocument(ctx, localPath, title)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[documentgenerator.Generate] 生成完成，文档内容长度: %d", len(result.Content))
	klog.V(6).Infof("[documentgenerator.Generate] 分析摘要: %s", result.AnalysisSummary)

	return result.Content, nil
}

// genDocument 执行文档生成链路，返回解析后的文档内容。
func (s *Service) genDocument(ctx context.Context, localPath string, title string) (*generationResult, error) {
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)

	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		s.factory,
		"document_generator_sequential_agent",
		"文档生成顺序执行 Agent - 分析代码并生成技术文档",
		agentDocumentGenerator,
	)

	if err != nil {
		return nil, fmt.Errorf("创建 Agent 失败: %w", err)
	}

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成一份技术文档。

仓库地址: %s
文档标题: %s

请按以下步骤执行：
1. 分析仓库代码，关注与文档类型相关的模块
2. 编写详细的技术文档，使用 Markdown 格式

请确保最终输出为有效的 Markdown 格式。`, localPath, title)

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
		return nil, ErrNoAgentOutput
	}

	klog.V(6).Infof("[documentgenerator.genDocument] Agent 输出原文: \n%s\n", lastContent)

	result, err := parseDocument(lastContent)
	if err != nil {
		klog.Errorf("[documentgenerator.genDocument] 解析文档生成结果失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[documentgenerator.genDocument] 执行成功，生成文档内容长度: %d", len(result.Content))
	return result, nil
}

// parseDocument 从 Agent 输出解析文档生成结果。
func parseDocument(content string) (*generationResult, error) {
	klog.V(6).Infof("[documentgenerator.parseDocument] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 从内容中提取 Markdown
	markdownContent := utils.ExtractMarkdown(content)
	if markdownContent == "" {
		klog.Warningf("[documentgenerator.parseDocument] 未能从内容中提取 Markdown")
		return nil, fmt.Errorf("%w: 未能从 Agent 输出中提取有效 Markdown", ErrEmptyDocumentContent)
	}

	// 校验结果
	if len(markdownContent) < 10 {
		return nil, fmt.Errorf("%w: 提取的 Markdown 内容过短", ErrEmptyDocumentContent)
	}

	result := &generationResult{
		Content:         markdownContent,
		AnalysisSummary: "文档生成完成",
	}

	klog.V(6).Infof("[documentgenerator.parseDocument] 解析成功，内容长度: %d", len(result.Content))
	return result, nil
}
