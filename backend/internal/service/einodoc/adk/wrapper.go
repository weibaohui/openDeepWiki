package adk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// ChatModelAgentWrapper 实现了 Eino ADK Agent 接口的包装器
// 将我们的业务逻辑 Agent 包装成符合 ADK 接口的 Agent
type ChatModelAgentWrapper struct {
	name        string                                           // Agent 名称
	description string                                           // Agent 描述
	state       *StateManager                                    // 状态管理器
	basePath    string                                           // 基础路径
	chatModel   model.ToolCallingChatModel                       // ChatModel 实例
	doExecute   func(context.Context, *StateManager, string) (*schema.Message, error) // 执行函数
}

// Info 返回 Agent 信息
// 实现 adk.Agent 接口
func (w *ChatModelAgentWrapper) Info() AgentInfo {
	return AgentInfo{
		Name:        w.name,
		Description: w.description,
	}
}

// Execute 执行 Agent
// 实现 adk.Agent 接口的核心方法
// 接收上下文和输入，返回执行结果
func (w *ChatModelAgentWrapper) Execute(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
	klog.V(6).Infof("[%s] Agent 开始执行", w.name)

	// 构建输入字符串
	inputStr := ""
	if input != nil && input.Message != nil {
		inputStr = input.Message.Content
	}

	// 执行具体的业务逻辑
	msg, err := w.doExecute(ctx, w.state, inputStr)
	if err != nil {
		klog.Errorf("[%s] Agent 执行失败: %v", w.name, err)
		return nil, err
	}

	klog.V(6).Infof("[%s] Agent 执行完成", w.name)

	// 返回 AgentOutput
	return &AgentOutput{
		Message: msg,
		Action:  nil, // 没有特殊 Action
	}, nil
}

// ==================== ADK 接口类型定义 ====================

// AgentInfo Agent 信息
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AgentInput Agent 输入
type AgentInput struct {
	Message *schema.Message `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// AgentOutput Agent 输出
type AgentOutput struct {
	Message *schema.Message `json:"message"`
	Action  *AgentAction    `json:"action,omitempty"`
}

// AgentAction Agent 动作
type AgentAction struct {
	Exit      *ExitAction      `json:"exit,omitempty"`
	BreakLoop *BreakLoopAction `json:"break_loop,omitempty"`
}

// ExitAction 退出动作
type ExitAction struct {
	From string `json:"from"`
}

// BreakLoopAction 跳出循环动作
type BreakLoopAction struct {
	From              string `json:"from"`
	Done              bool   `json:"done"`
	CurrentIterations int    `json:"current_iterations"`
}

// Agent 接口定义
// 与 Eino ADK 的 Agent 接口保持一致
type Agent interface {
	Info() AgentInfo
	Execute(ctx context.Context, input *AgentInput) (*AgentOutput, error)
}

// ==================== SequentialAgent 实现 ====================

// SequentialAgent 顺序执行多个子 Agent
type SequentialAgent struct {
	name      string
	description string
	subAgents []Agent
}

// SequentialAgentConfig SequentialAgent 配置
type SequentialAgentConfig struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SubAgents   []Agent `json:"sub_agents"`
}

// NewSequentialAgent 创建新的 SequentialAgent
func NewSequentialAgent(ctx context.Context, config *SequentialAgentConfig) (Agent, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	klog.V(6).Infof("[SequentialAgent] 创建顺序 Agent: name=%s, subAgents=%d", config.Name, len(config.SubAgents))

	return &SequentialAgent{
		name:        config.Name,
		description: config.Description,
		subAgents:   config.SubAgents,
	}, nil
}

// Info 返回 Agent 信息
func (s *SequentialAgent) Info() AgentInfo {
	return AgentInfo{
		Name:        s.name,
		Description: s.description,
	}
}

// Execute 顺序执行所有子 Agent
func (s *SequentialAgent) Execute(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
	klog.V(6).Infof("[SequentialAgent] 开始执行: %s", s.name)

	var lastOutput *AgentOutput

	for i, agent := range s.subAgents {
		info := agent.Info()
		klog.V(6).Infof("[SequentialAgent] 执行子 Agent [%d/%d]: %s", i+1, len(s.subAgents), info.Name)

		// 准备输入：第一个 Agent 使用原始输入，后续 Agent 使用上一个 Agent 的输出
		agentInput := input
		if lastOutput != nil && lastOutput.Message != nil {
			agentInput = &AgentInput{
				Message: lastOutput.Message,
				Context: input.Context,
			}
		}

		output, err := agent.Execute(ctx, agentInput)
		if err != nil {
			klog.Errorf("[SequentialAgent] 子 Agent %s 执行失败: %v", info.Name, err)
			return nil, fmt.Errorf("agent %s failed: %w", info.Name, err)
		}

		lastOutput = output

		// 检查是否有退出动作
		if output.Action != nil {
			if output.Action.Exit != nil {
				klog.V(6).Infof("[SequentialAgent] 收到退出信号，终止执行")
				return output, nil
			}
		}

		klog.V(6).Infof("[SequentialAgent] 子 Agent %s 执行完成", info.Name)
	}

	klog.V(6).Infof("[SequentialAgent] 所有子 Agent 执行完成")
	return lastOutput, nil
}

// ==================== Runner 实现 ====================

// Runner ADK Runner
type Runner struct {
	agent Agent
}

// RunnerConfig Runner 配置
type RunnerConfig struct {
	Agent Agent `json:"agent"`
}

// NewRunner 创建新的 Runner
func NewRunner(ctx context.Context, config RunnerConfig) *Runner {
	klog.V(6).Infof("[Runner] 创建 Runner")
	return &Runner{
		agent: config.Agent,
	}
}

// Query 执行查询
// 返回事件迭代器
func (r *Runner) Query(ctx context.Context, input string) *EventIterator {
	klog.V(6).Infof("[Runner] 开始 Query: input=%s", truncate(input, 100))

	events := make(chan *RunnerEvent, 10)

	go func() {
		defer close(events)

		// 创建输入
		agentInput := &AgentInput{
			Message: &schema.Message{
				Role:    schema.User,
				Content: input,
			},
		}

		// 执行 Agent
		output, err := r.agent.Execute(ctx, agentInput)

		if err != nil {
			events <- &RunnerEvent{
				AgentName: r.agent.Info().Name,
				Err:       err,
			}
			return
		}

		// 发送输出事件
		if output != nil && output.Message != nil {
			events <- &RunnerEvent{
				AgentName: r.agent.Info().Name,
				Output: &AgentOutput{
					Message: output.Message,
					Action:  output.Action,
				},
			}
		}
	}()

	return &EventIterator{
		ch: events,
	}
}

// RunnerEvent Runner 事件
type RunnerEvent struct {
	AgentName string       `json:"agent_name"`
	Output    *AgentOutput `json:"output,omitempty"`
	Err       error        `json:"err,omitempty"`
}

// EventIterator 事件迭代器
type EventIterator struct {
	ch <-chan *RunnerEvent
}

// Next 获取下一个事件
// 如果没有更多事件，返回 false
func (it *EventIterator) Next() (*RunnerEvent, bool) {
	event, ok := <-it.ch
	return event, ok
}

// ==================== 辅助函数 ====================

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\": \"%s\"}", err.Error())
	}
	return string(data)
}
