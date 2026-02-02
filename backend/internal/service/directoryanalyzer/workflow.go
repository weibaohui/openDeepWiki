package directoryanalyzer

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
	AgentTaskGenerator = "TaskGenerator" // 任务生成 Agent
	AgentTaskValidator = "TaskValidator" // 任务校验 Agent
)

// TaskGeneratorWorkflow 任务生成工作流
// 使用 SequentialAgent 顺序执行 TaskGenerator 和 TaskValidator
type TaskGeneratorWorkflow struct {
	cfg             *config.Config
	sequentialAgent adk.ResumableAgent
	factory         *adkagents.AgentFactory
}

// NewTaskGeneratorWorkflow 创建新的任务生成工作流
func NewTaskGeneratorWorkflow(cfg *config.Config) (*TaskGeneratorWorkflow, error) {
	factory, err := adkagents.NewAgentFactory(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent factory: %w", err)
	}
	return &TaskGeneratorWorkflow{
		cfg:     cfg,
		factory: factory,
	}, nil
}

// Build 构建 SequentialAgent
func (w *TaskGeneratorWorkflow) Build(ctx context.Context) error {
	sequentialAgent, err := adkagents.BuildSequentialAgent(
		ctx,
		w.factory,
		"TaskGeneratorSequentialAgent",
		"任务生成顺序执行 Agent - 先生成任务列表，再校验修正",
		AgentTaskGenerator,
		AgentTaskValidator,
	)
	if err != nil {
		return fmt.Errorf("failed to create sequential agent: %w", err)
	}
	w.sequentialAgent = sequentialAgent
	return nil
}

// Run 执行 Workflow
// ctx: 上下文
// localPath: 仓库本地路径
// 返回: TaskGenerationResult 或错误
func (w *TaskGeneratorWorkflow) Run(ctx context.Context, localPath string) (*TaskGenerationResult, error) {
	klog.V(6).Infof("[TaskGeneratorWorkflow.Run] 开始执行: localPath=%s", localPath)

	// 构建 Workflow（如果还没有构建）
	if w.sequentialAgent == nil {
		if err := w.Build(ctx); err != nil {
			return nil, err
		}
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: w.sequentialAgent,
	})

	// 准备初始消息
	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库，并生成需要的技术分析任务列表。

仓库地址: %s

请按以下步骤执行：
1. 分析仓库目录结构，识别项目类型和技术栈
2. 根据项目特征生成初步的任务列表
3. 校验并修正任务列表，确保完整性和合理性

请确保最终输出格式为有效的 JSON。`, localPath)

	// 设置会话值，供 Agent 使用
	adk.AddSessionValue(ctx, "local_path", localPath)

	// 执行 Workflow
	iter := runner.Run(ctx, []adk.Message{
		{
			Role:    schema.User,
			Content: initialMessage,
		},
	})

	// 收集执行结果
	var lastContent string
	stepCount := 0

	// 循环处理每个 agent 的执行结果
	for {
		select {
		case <-ctx.Done():
			klog.Warningf("[TaskGeneratorWorkflow.Run] 上下文被取消: %v", ctx.Err())
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			// 检查是否是迭代次数超限的错误
			if adkagents.IsMaxIterationsError(event.Err) {
				klog.Warningf("[TaskGeneratorWorkflow.Run] 检测到迭代次数超限错误，尝试使用最后的内容: %v", event.Err)
				// 尝试使用 lastContent 解析结果
				if lastContent != "" {
					result, err := ParseTaskGenerationResult(lastContent)
					if err == nil {
						return result, nil
					}
				}
			}
			klog.Errorf("[TaskGeneratorWorkflow.Run] Agent 执行出错: %v", event.Err)
			return nil, fmt.Errorf("agent execution failed: %w", event.Err)
		}

		stepCount++

		// 记录每个 Agent 的输出
		if event.Output != nil && event.Output.MessageOutput != nil {
			content := event.Output.MessageOutput.Message.Content
			lastContent = content

			klog.V(6).Infof("[TaskGeneratorWorkflow.Run] 步骤 %d [%s] 完成, 内容长度: %d",
				stepCount, event.AgentName, len(content))
		}

		// 处理 Action
		if event.Action != nil && event.Action.Exit {
			klog.V(6).Infof("[TaskGeneratorWorkflow.Run] 收到退出信号")
			break
		}
	}

	klog.V(6).Infof("[TaskGeneratorWorkflow.Run] 所有步骤执行完成: steps=%d", stepCount)

	// 解析最终结果
	if lastContent == "" {
		return nil, fmt.Errorf("no content generated from workflow")
	}

	result, err := ParseTaskGenerationResult(lastContent)
	if err != nil {
		klog.Errorf("[TaskGeneratorWorkflow.Run] 解析结果失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[TaskGeneratorWorkflow.Run] 执行成功，生成任务数: %d", len(result.Tasks))
	return result, nil
}
