package documentgenerator

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"k8s.io/klog/v2"
)

// Agent 名称常量
const (
	agentDocumentGenerator = "DocumentGenerator" // 文档生成 Agent
)

// 错误定义
var (
	ErrAgentExecutionFailed = errors.New("Agent 执行失败")
	ErrNoAgentOutput        = errors.New("Agent 未产生任何输出内容")
)

// DocumentGeneratorWorkflow 文档生成工作流
type DocumentGeneratorWorkflow struct {
	cfg     *config.Config
	factory *adkagents.AgentFactory
}

// NewDocumentGeneratorWorkflow 创建新的文档生成工作流
func NewDocumentGeneratorWorkflow(cfg *config.Config) (*DocumentGeneratorWorkflow, error) {
	klog.V(6).Infof("[DocumentGeneratorWorkflow.New] 开始创建文档生成工作流")

	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		klog.Errorf("[DocumentGeneratorWorkflow.New] 创建 AgentFactory 失败: %v", err)
		return nil, fmt.Errorf("创建 AgentFactory 失败: %w", err)
	}

	return &DocumentGeneratorWorkflow{
		cfg:     cfg,
		factory: factory,
	}, nil
}

// Run 执行 Workflow，返回解析后的文档生成结果
func (w *DocumentGeneratorWorkflow) Run(ctx context.Context, localPath string, title string) (string, error) {
	klog.V(6).Infof("[DocumentGeneratorWorkflow.Run] 开始执行: localPath=%s, title=%s", localPath, title)

	// 添加会话值
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "document_title", title)

	// 构建顺序执行 Agent
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		w.factory,
		"document_generator_sequential_agent",
		"文档生成顺序执行 Agent - 分析代码并生成技术文档",
		agentDocumentGenerator,
	)

	if err != nil {
		klog.Errorf("[DocumentGeneratorWorkflow.Run] 创建 Agent 失败: %v", err)
		return "", fmt.Errorf("创建 Agent 失败: %w", err)
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
		return "", fmt.Errorf("%w: %w", ErrAgentExecutionFailed, err)
	}

	if lastContent == "" {
		return "", ErrNoAgentOutput
	}

	klog.V(6).Infof("[DocumentGeneratorWorkflow.Run] Agent 输出原文: \n%s\n", lastContent)

	result, err := ParseDocumentGenerationResult(lastContent)
	if err != nil {
		klog.Errorf("[DocumentGeneratorWorkflow.Run] 解析文档生成结果失败: %v", err)
		return "", err
	}

	klog.V(6).Infof("[DocumentGeneratorWorkflow.Run] 执行成功，生成文档内容长度: %d", len(result))
	return result, nil
}
