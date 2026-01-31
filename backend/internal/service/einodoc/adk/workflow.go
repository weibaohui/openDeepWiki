package adk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	"k8s.io/klog/v2"
)

// RepoDocSequentialWorkflow 基于 SequentialAgent 的仓库文档生成工作流
// 使用 Eino ADK 的 SequentialAgent 模式，将各个步骤封装为独立的 Agent
// 按照预定顺序依次执行：RepoInitializer -> Architect -> Explorer -> Writer -> Editor
type RepoDocSequentialWorkflow struct {
	sequentialAgent Agent           // SequentialAgent 实例
	state           *StateManager   // 状态管理器
	basePath        string          // 仓库存储基础路径
	chatModel       model.ToolCallingChatModel // ChatModel 实例
}

// NewRepoDocSequentialWorkflow 创建新的 Sequential Workflow
// basePath: 仓库存储的基础路径
// chatModel: Eino ChatModel 实例，用于 LLM 调用
// 返回: 配置好的 Workflow 实例或错误
func NewRepoDocSequentialWorkflow(basePath string, chatModel model.ToolCallingChatModel) (*RepoDocSequentialWorkflow, error) {
	klog.V(6).Infof("[NewRepoDocSequentialWorkflow] 开始创建 Sequential Workflow: basePath=%s", basePath)

	return &RepoDocSequentialWorkflow{
		basePath:  basePath,
		chatModel: chatModel,
	}, nil
}

// Build 构建 SequentialAgent
// 根据输入创建所有子 Agent 并组装成 SequentialAgent
// 必须在 Run 之前调用
func (w *RepoDocSequentialWorkflow) Build(ctx context.Context, repoURL string) error {
	klog.V(6).Infof("[RepoDocSequentialWorkflow.Build] 开始构建 Workflow: repoURL=%s", repoURL)

	// 创建状态管理器
	w.state = NewStateManager(repoURL, "")

	// 创建 Agent 工厂
	factory := NewAgentFactory(w.chatModel, w.basePath)

	// 创建各个子 Agent
	agents := make([]Agent, 0, 5)

	// 1. RepoInitializer Agent - 仓库初始化
	initializer, err := factory.CreateRepoInitializerAgent(w.state)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 RepoInitializer 失败: %v", err)
		return fmt.Errorf("failed to create repo initializer: %w", err)
	}
	agents = append(agents, initializer)

	// 2. Architect Agent - 架构分析
	architect, err := factory.CreateArchitectAgent(w.state)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 Architect 失败: %v", err)
		return fmt.Errorf("failed to create architect: %w", err)
	}
	agents = append(agents, architect)

	// 3. Explorer Agent - 代码探索
	explorer, err := factory.CreateExplorerAgent(w.state)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 Explorer 失败: %v", err)
		return fmt.Errorf("failed to create explorer: %w", err)
	}
	agents = append(agents, explorer)

	// 4. Writer Agent - 文档撰写
	writer, err := factory.CreateWriterAgent(w.state)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 Writer 失败: %v", err)
		return fmt.Errorf("failed to create writer: %w", err)
	}
	agents = append(agents, writer)

	// 5. Editor Agent - 文档编辑
	editor, err := factory.CreateEditorAgent(w.state)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 Editor 失败: %v", err)
		return fmt.Errorf("failed to create editor: %w", err)
	}
	agents = append(agents, editor)

	// 创建 SequentialAgent
	config := &SequentialAgentConfig{
		Name:        "RepoDocSequentialAgent",
		Description: "仓库文档生成顺序执行 Agent - 按顺序执行初始化、分析、探索、撰写、编辑",
		SubAgents:   agents,
	}

	sequentialAgent, err := NewSequentialAgent(ctx, config)
	if err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Build] 创建 SequentialAgent 失败: %v", err)
		return fmt.Errorf("failed to create sequential agent: %w", err)
	}

	w.sequentialAgent = sequentialAgent

	klog.V(6).Infof("[RepoDocSequentialWorkflow.Build] Workflow 构建完成: agents=%d", len(agents))
	return nil
}

// Run 执行 Workflow
// 构建并执行 SequentialAgent，生成文档
// ctx: 上下文
// repoURL: 仓库 Git URL
// 返回: RepoDocResult 或错误
func (w *RepoDocSequentialWorkflow) Run(ctx context.Context, repoURL string) (*einodoc.RepoDocResult, error) {
	klog.V(6).Infof("[RepoDocSequentialWorkflow.Run] 开始执行 Workflow: repoURL=%s", repoURL)

	// 构建 Workflow
	if err := w.Build(ctx, repoURL); err != nil {
		klog.Errorf("[RepoDocSequentialWorkflow.Run] 构建 Workflow 失败: %v", err)
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	// 创建 Runner
	runner := NewRunner(ctx, RunnerConfig{
		Agent: w.sequentialAgent,
	})

	// 准备输入
	input := fmt.Sprintf(`{"repo_url": "%s"}`, repoURL)

	klog.V(6).Infof("[RepoDocSequentialWorkflow.Run] 启动 SequentialAgent 执行")

	// 执行 Workflow
	iter := runner.Query(ctx, input)

	// 收集执行结果
	stepCount := 0
	var lastEvent *RunnerEvent

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			klog.Errorf("[RepoDocSequentialWorkflow.Run] Agent 执行出错: %v", event.Err)
			return nil, fmt.Errorf("agent execution failed: %w", event.Err)
		}

		if event.Output != nil && event.Output.Message != nil {
			stepCount++
			klog.V(6).Infof("[RepoDocSequentialWorkflow.Run] 步骤 %d [%s] 完成: %s",
				stepCount,
				event.AgentName,
				truncate(event.Output.Message.Content, 100))
			lastEvent = event
		}
	}

	klog.V(6).Infof("[RepoDocSequentialWorkflow.Run] 所有步骤执行完成: steps=%d", stepCount)

	// 构建最终结果
	if lastEvent != nil && w.state != nil {
		result := w.state.BuildResult()
		klog.V(6).Infof("[RepoDocSequentialWorkflow.Run] 文档生成成功: length=%d, sections=%d",
			len(result.Document), result.SectionsCount)
		return result, nil
	}

	return nil, fmt.Errorf("workflow execution failed: no result")
}

// RunWithProgress 执行 Workflow 并返回进度事件
// 适用于需要实时展示进度的场景
func (w *RepoDocSequentialWorkflow) RunWithProgress(ctx context.Context, repoURL string) (<-chan *WorkflowProgressEvent, error) {
	klog.V(6).Infof("[RepoDocSequentialWorkflow.RunWithProgress] 开始执行 Workflow: repoURL=%s", repoURL)

	// 构建 Workflow
	if err := w.Build(ctx, repoURL); err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	// 创建进度事件通道
	progressCh := make(chan *WorkflowProgressEvent, 10)

	// 异步执行
	go func() {
		defer close(progressCh)

		// 创建 Runner
		runner := NewRunner(ctx, RunnerConfig{
			Agent: w.sequentialAgent,
		})

		// 准备输入
		input := fmt.Sprintf(`{"repo_url": "%s"}`, repoURL)

		// 执行 Workflow
		iter := runner.Query(ctx, input)

		stepCount := 0
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			stepCount++

			if event.Err != nil {
				progressCh <- &WorkflowProgressEvent{
					Step:      stepCount,
					AgentName: event.AgentName,
					Status:    WorkflowStatusError,
					Error:     event.Err,
				}
				return
			}

			status := WorkflowStatusCompleted
			content := ""
			if event.Output != nil && event.Output.Message != nil {
				content = event.Output.Message.Content
			}

			progressCh <- &WorkflowProgressEvent{
				Step:      stepCount,
				AgentName: event.AgentName,
				Status:    status,
				Content:   content,
			}
		}

		// 发送完成事件
		if w.state != nil {
			result := w.state.BuildResult()
			progressCh <- &WorkflowProgressEvent{
				Step:      stepCount + 1,
				AgentName: "FinalResult",
				Status:    WorkflowStatusFinished,
				Result:    result,
			}
		}
	}()

	return progressCh, nil
}

// GetState 获取当前状态
func (w *RepoDocSequentialWorkflow) GetState() *StateManager {
	return w.state
}

// ==================== Workflow 进度事件 ====================

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	// WorkflowStatusRunning 正在执行
	WorkflowStatusRunning WorkflowStatus = "running"
	// WorkflowStatusCompleted 步骤完成
	WorkflowStatusCompleted WorkflowStatus = "completed"
	// WorkflowStatusError 执行出错
	WorkflowStatusError WorkflowStatus = "error"
	// WorkflowStatusFinished 全部完成
	WorkflowStatusFinished WorkflowStatus = "finished"
)

// WorkflowProgressEvent 工作流进度事件
type WorkflowProgressEvent struct {
	Step      int                `json:"step"`       // 步骤序号
	AgentName string             `json:"agent_name"` // Agent 名称
	Status    WorkflowStatus     `json:"status"`     // 状态
	Content   string             `json:"content"`    // 内容摘要
	Error     error              `json:"error,omitempty"` // 错误信息
	Result    *einodoc.RepoDocResult `json:"result,omitempty"` // 最终结果
}

// ==================== 便捷构建函数 ====================

// BuildWorkflowInput 构建 Workflow 输入
func BuildWorkflowInput(repoURL string) string {
	input := WorkflowInput{
		RepoURL: repoURL,
	}
	data, _ := json.Marshal(input)
	return string(data)
}

// ParseWorkflowOutput 解析 Workflow 输出
func ParseWorkflowOutput(content string) (*WorkflowOutput, error) {
	var output WorkflowOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse workflow output: %w", err)
	}
	return &output, nil
}

// ==================== 调试和工具函数 ====================

// WorkflowInfo Workflow 信息
type WorkflowInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Agents      []string `json:"agents"`
}

// GetWorkflowInfo 获取 Workflow 信息
func (w *RepoDocSequentialWorkflow) GetWorkflowInfo() *WorkflowInfo {
	if w.sequentialAgent == nil {
		return &WorkflowInfo{
			Name:        "RepoDocSequentialWorkflow",
			Description: "基于 SequentialAgent 的仓库文档生成工作流（未构建）",
			Agents:      []string{},
		}
	}

	info := w.sequentialAgent.Info()

	// 获取子 Agent 列表
	agents := make([]string, 0)
	if seqAgent, ok := w.sequentialAgent.(*SequentialAgent); ok {
		for _, agent := range seqAgent.subAgents {
			agents = append(agents, agent.Info().Name)
		}
	}

	return &WorkflowInfo{
		Name:        info.Name,
		Description: info.Description,
		Agents:      agents,
	}
}

// ToJSON 将 Workflow 信息转换为 JSON
func (info *WorkflowInfo) ToJSON() string {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ==================== 兼容原有工具的辅助函数 ====================

// GenerateRepoDirName 从 repo URL 生成目录名
func GenerateRepoDirName(repoURL string) string {
	return tools.GenerateRepoDirName(repoURL)
}


