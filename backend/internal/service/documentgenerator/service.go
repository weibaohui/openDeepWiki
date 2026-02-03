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
	agentGen = "document_generator" // 文档生成 Agent
)

// 错误定义
var (
	ErrInvalidLocalPath     = errors.New("invalid local path")
	ErrAgentExecutionFailed = errors.New("agent execution failed")
	ErrEmptyContent         = errors.New("empty content")
	ErrNoAgentOutput        = errors.New("no agent output")
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
	klog.V(6).Infof("[dgen.New] creating document generator service")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[dgen.New] create AgentFactory failed: %v", err)
		return nil, fmt.Errorf("create AgentFactory failed: %w", err)
	}

	return &Service{
		factory: factory,
	}, nil
}

// Generate 分析仓库代码并生成文档。
func (s *Service) Generate(ctx context.Context, localPath string, title string) (string, error) {
	if localPath == "" {
		return "", fmt.Errorf("%w: local path is empty", ErrInvalidLocalPath)
	}
	if title == "" {
		return "", fmt.Errorf("%w: title is empty", ErrInvalidLocalPath)
	}

	klog.V(6).Infof("[dgen.Generate] generating document: localPath=%s, title=%s", localPath, title)

	result, err := s.genDocument(ctx, localPath, title)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	klog.V(6).Infof("[dgen.Generate] generation complete, content length: %d", len(result.Content))
	klog.V(6).Infof("[dgen.Generate] analysis summary: %s", result.AnalysisSummary)

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
		"document generator sequential agent - analyze code and generate documentation",
		agentGen,
	)

	if err != nil {
		return nil, fmt.Errorf("create agent failed: %w", err)
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
		return nil, fmt.Errorf("agent execution error: %w", err)
	}

	if lastContent == "" {
		return nil, ErrNoAgentOutput
	}

	klog.V(6).Infof("[dgen.genDoc] agent output: \n%s\n", lastContent)

	result, err := parseDocument(lastContent)
	if err != nil {
		klog.Errorf("[dgen.genDoc] parse document result failed: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[dgen.genDoc] execution success, content length: %d", len(result.Content))
	return result, nil
}

// parseDocument 从 Agent 输出解析文档生成结果。
func parseDocument(content string) (*generationResult, error) {
	klog.V(6).Infof("[dgen.parseDoc] parsing agent output, content length: %d", len(content))

	// 从内容中提取 Markdown
	markdown := utils.ExtractMarkdown(content)
	if markdown == "" {
		klog.Warningf("[dgen.parseDoc] failed to extract markdown from content")
		return nil, fmt.Errorf("%w: extract markdown from agent output failed", ErrEmptyContent)
	}

	// 校验结果
	if len(markdown) < 10 {
		return nil, fmt.Errorf("%w: extracted markdown too short", ErrEmptyContent)
	}

	result := &generationResult{
		Content:         markdown,
		AnalysisSummary: "document generation complete",
	}

	klog.V(6).Infof("[dgen.parseDoc] parse success, content length: %d", len(result.Content))
	return result, nil
}
