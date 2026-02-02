package adkagents

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/weibaohui/opendeepwiki/backend/config"
)

// AgentFactory 负责创建各种子 Agent
// 使用 adkagents.Manager 管理基础 Agent 的加载和创建
type AgentFactory struct {
	Manager  *Manager
	basePath string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(cfg *config.Config) (*AgentFactory, error) {

	manager, err := GetOrCreateInstance(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create adkagents manager: %w", err)
	}

	return &AgentFactory{
		Manager: manager,
	}, nil
}

// GetAgent 获取指定名称的基础 Agent
// 这是获取基础 Agent 的推荐方式
func (f *AgentFactory) GetAgent(name string) (adk.Agent, error) {
	return f.Manager.GetAgent(name)
}

// Stop 停止 AgentFactory，释放资源
func (f *AgentFactory) Stop() {
	if f.Manager != nil {
		f.Manager.Stop()
	}
}

// BuildSequentialAgent 根据 Agent 名称列表创建 SequentialAgent
// ctx: 上下文
// factory: Agent 工厂
// name: SequentialAgent 名称
// description: SequentialAgent 描述
// agentNames: 需要按顺序执行的子 Agent 名称列表
// 返回: 构建好的 ResumableAgent 或错误
func BuildSequentialAgent(
	ctx context.Context,
	factory *AgentFactory,
	name string,
	description string,
	agentNames ...string,
) (adk.ResumableAgent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if factory == nil {
		return nil, fmt.Errorf("agent factory 为空")
	}
	if len(agentNames) == 0 {
		return nil, fmt.Errorf("agentNames 为空")
	}

	subAgents := make([]adk.Agent, 0, len(agentNames))
	for _, agentName := range agentNames {
		agent, err := factory.GetAgent(agentName)
		if err != nil {
			return nil, fmt.Errorf("获取 Agent 失败: name=%s, err=%w", agentName, err)
		}
		subAgents = append(subAgents, agent)
	}

	cfg := &adk.SequentialAgentConfig{
		Name:        name,
		Description: description,
		SubAgents:   subAgents,
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 SequentialAgent 失败: %w", err)
	}

	return sequentialAgent, nil
}

// IsMaxIterationsError 判断错误是否为“超过最大迭代次数”类错误
// 说明: Eino/ADK 在不同场景下报错文本可能不同，这里做兼容匹配
func IsMaxIterationsError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "exceeds max iterations") || strings.Contains(errMsg, "max iterations")
}

// RunAgentToLastContent 运行指定 Agent，返回最后一次输出的内容
// ctx: 上下文
// agent: 需要运行的 Agent
// messages: 初始消息列表
// 返回: lastContent（可能为空）、error（若中途出错）
func RunAgentToLastContent(ctx context.Context, agent adk.Agent, messages []adk.Message) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if agent == nil {
		return "", fmt.Errorf("agent 为空")
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Run(ctx, messages)

	var lastContent string
	for {
		select {
		case <-ctx.Done():
			return lastContent, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return lastContent, event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			lastContent = event.Output.MessageOutput.Message.Content
		}
		if event.Action != nil && event.Action.Exit {
			break
		}
	}

	return lastContent, nil
}
