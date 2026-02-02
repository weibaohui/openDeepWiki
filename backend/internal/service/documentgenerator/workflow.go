package documentgenerator

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"k8s.io/klog/v2"
)

// Agent 名称常量
const (
	AgentDocumentGenerator = "DocumentGenerator" // 文档生成 Agent
)

// DocumentGeneratorWorkflow 文档生成工作流
type DocumentGeneratorWorkflow struct {
	cfg             *config.Config
	agent           adk.ResumableAgent
	factory         *adkagents.AgentFactory
}

// NewDocumentGeneratorWorkflow 创建新的文档生成工作流
func NewDocumentGeneratorWorkflow(cfg *config.Config) (*DocumentGeneratorWorkflow, error) {
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent factory: %w", err)
	}
	return &DocumentGeneratorWorkflow{
		cfg:     cfg,
		factory: factory,
	}, nil
}

// Build 构建 Agent
func (w *DocumentGeneratorWorkflow) Build(ctx context.Context) error {
	// 使用 SequentialAgent 包装，即使只有一个 Agent，也方便未来扩展
	agent, err := adkagents.BuildSequentialAgent(
		ctx,
		w.factory,
		"DocumentGeneratorSequentialAgent",
		"文档生成 Agent - 分析代码并生成技术文档",
		AgentDocumentGenerator,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	w.agent = agent
	return nil
}

// Run 执行 Workflow
func (w *DocumentGeneratorWorkflow) Run(ctx context.Context, localPath string, taskType string) (*DocumentGenerationResult, error) {
	klog.V(6).Infof("[DocumentGeneratorWorkflow.Run] 开始执行: localPath=%s, taskType=%s", localPath, taskType)

	if w.agent == nil {
		if err := w.Build(ctx); err != nil {
			return nil, err
		}
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: w.agent,
	})

	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成一份技术文档。

仓库地址: %s
文档类型: %s

请按以下步骤执行：
1. 分析仓库代码，关注与文档类型相关的模块
2. 编写详细的技术文档，使用 Markdown 格式
3. 将结果封装在 JSON 中返回，格式如下：
{
  "content": "生成的 Markdown 内容",
  "analysis_summary": "分析过程摘要"
}
`, localPath, taskType)

	adk.AddSessionValue(ctx, "local_path", localPath)

	iter := runner.Run(ctx, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})

	var lastContent string
	stepCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			if adkagents.IsMaxIterationsError(event.Err) {
				klog.Warningf("[DocumentGeneratorWorkflow.Run] 检测到迭代次数超限错误，尝试使用最后的内容: %v", event.Err)
				if lastContent != "" {
					result, err := ParseDocumentGenerationResult(lastContent)
					if err == nil {
						return result, nil
					}
				}
			}
			klog.Errorf("[DocumentGeneratorWorkflow.Run] Agent 执行出错: %v", event.Err)
			return nil, fmt.Errorf("agent execution failed: %w", event.Err)
		}

		stepCount++

		if event.Output != nil && event.Output.MessageOutput != nil {
			content := event.Output.MessageOutput.Message.Content
			lastContent = content
			klog.V(6).Infof("[DocumentGeneratorWorkflow.Run] 步骤 %d [%s] 完成, 内容长度: %d",
				stepCount, event.AgentName, len(content))
		}

		if event.Action != nil && event.Action.Exit {
			break
		}
	}

	if lastContent == "" {
		return nil, fmt.Errorf("no content generated from workflow")
	}

	return ParseDocumentGenerationResult(lastContent)
}
