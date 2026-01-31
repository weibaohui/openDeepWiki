package adk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	etools "github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	"k8s.io/klog/v2"
)

// RepoDocWorkflow 基于 Eino ADK 的仓库文档生成工作流
// 使用原生的 SequentialAgent 和 Runner
type RepoDocWorkflow struct {
	sequentialAgent adk.ResumableAgent // 使用原生的 SequentialAgent
	state           *einodoc.RepoDocState
	basePath        string
	chatModel       model.ToolCallingChatModel
	factory         *AgentFactory
}

// NewRepoDocWorkflow 创建新的 ADK Workflow
// basePath: 仓库存储的基础路径
// chatModel: Eino ChatModel 实例
func NewRepoDocWorkflow(basePath string, chatModel model.ToolCallingChatModel) (*RepoDocWorkflow, error) {
	klog.V(6).Infof("[NewRepoDocWorkflow] 创建 Workflow: basePath=%s", basePath)

	return &RepoDocWorkflow{
		basePath:  basePath,
		chatModel: chatModel,
		factory:   NewAgentFactory(chatModel, basePath),
	}, nil
}

// Build 构建 SequentialAgent
// 创建所有子 Agent 并组装成 SequentialAgent
func (w *RepoDocWorkflow) Build(ctx context.Context) error {
	klog.V(6).Infof("[RepoDocWorkflow.Build] 开始构建 Workflow")

	// 使用工厂创建 SequentialAgent
	sequentialAgent, err := w.factory.CreateSequentialAgent()
	if err != nil {
		klog.Errorf("[RepoDocWorkflow.Build] 创建 SequentialAgent 失败: %v", err)
		return fmt.Errorf("failed to create sequential agent: %w", err)
	}

	w.sequentialAgent = sequentialAgent

	klog.V(6).Infof("[RepoDocWorkflow.Build] Workflow 构建完成")
	return nil
}

// Run 执行 Workflow
// ctx: 上下文
// localPath: 仓库本地路径
// 返回: RepoDocResult 或错误
func (w *RepoDocWorkflow) Run(ctx context.Context, localPath string) (*einodoc.RepoDocResult, error) {
	klog.V(6).Infof("[RepoDocWorkflow.Run] 开始执行: localPath=%s", localPath)

	// 初始化状态
	w.state = einodoc.NewRepoDocState(localPath, "")

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
	initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库并生成技术文档。

仓库地址: %s

请按以下步骤执行：
1. 分析仓库类型和技术栈，生成文档大纲
2. 深入探索代码结构
3. 为每个小节生成文档内容
4. 组装最终文档

请确保每个步骤都完整执行。`, localPath)

	// 设置会话值，供 Agent 使用
	adk.AddSessionValue(ctx, "local_path", localPath)
	adk.AddSessionValue(ctx, "base_path", w.basePath)
	adk.AddSessionValue(ctx, "target_dir", etools.GenerateRepoDirName(localPath))

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

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			klog.Errorf("[RepoDocWorkflow.Run] Agent 执行出错: %v", event.Err)
			return nil, fmt.Errorf("agent execution failed: %w", event.Err)
		}

		stepCount++

		// 记录每个 Agent 的输出
		if event.Output != nil && event.Output.MessageOutput != nil {
			content := event.Output.MessageOutput.Message.Content
			lastContent = content

			klog.V(6).Infof("[RepoDocWorkflow.Run] 步骤 %d [%s] 完成, 内容长度: %d",
				stepCount, event.AgentName, len(content))

			// 根据 Agent 名称处理输出
			switch event.AgentName {
			case AgentArchitect:
				w.processArchitectOutput(content)
			case AgentWriter:
				w.processWriterOutput(content)
			case AgentEditor:
				w.processEditorOutput(content)
			}
		}

		// 处理 Action
		if event.Action != nil && event.Action.Exit {
			klog.V(6).Infof("[RepoDocWorkflow.Run] 收到退出信号")
			break
		}
	}

	klog.V(6).Infof("[RepoDocWorkflow.Run] 所有步骤执行完成: steps=%d", stepCount)

	// 构建最终结果
	result := w.buildResult(localPath, lastContent)
	return result, nil
}

// RunWithProgress 执行 Workflow 并返回进度事件
// 适用于需要实时展示进度的场景
func (w *RepoDocWorkflow) RunWithProgress(ctx context.Context, localPath string) (<-chan *WorkflowProgressEvent, error) {
	klog.V(6).Infof("[RepoDocWorkflow.RunWithProgress] 开始执行: localPath=%s", localPath)

	// 初始化状态
	w.state = einodoc.NewRepoDocState(localPath, "")

	// 构建 Workflow
	if w.sequentialAgent == nil {
		if err := w.Build(ctx); err != nil {
			return nil, err
		}
	}

	// 创建进度事件通道
	progressCh := make(chan *WorkflowProgressEvent, 10)

	// 异步执行
	go func() {
		defer close(progressCh)

		// 创建 Runner
		runner := adk.NewRunner(ctx, adk.RunnerConfig{
			Agent: w.sequentialAgent,
		})

		// 设置会话值
		adk.AddSessionValue(ctx, "repo_path", localPath)
		adk.AddSessionValue(ctx, "base_path", w.basePath)
		adk.AddSessionValue(ctx, "target_dir", etools.GenerateRepoDirName(localPath))

		// 执行 Workflow
		initialMessage := fmt.Sprintf(`请帮我分析这个代码仓库并生成技术文档。

仓库地址: %s`, localPath)

		iter := runner.Run(ctx, []adk.Message{
			{
				Role:    schema.User,
				Content: initialMessage,
			},
		})

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

			status := WorkflowStatusRunning
			content := ""

			if event.Output != nil && event.Output.MessageOutput != nil {
				content = event.Output.MessageOutput.Message.Content
				status = WorkflowStatusCompleted

				// 处理特定 Agent 的输出
				switch event.AgentName {
				case AgentArchitect:
					w.processArchitectOutput(content)
				case AgentWriter:
					w.processWriterOutput(content)
				case AgentEditor:
					w.processEditorOutput(content)
				}
			}

			progressCh <- &WorkflowProgressEvent{
				Step:      stepCount,
				AgentName: event.AgentName,
				Status:    status,
				Content:   truncate(content, 200),
			}

			// 检查退出动作
			if event.Action != nil && event.Action.Exit {
				break
			}
		}

		// 发送完成事件
		result := w.buildResult(localPath, "")
		progressCh <- &WorkflowProgressEvent{
			Step:      stepCount + 1,
			AgentName: "FinalResult",
			Status:    WorkflowStatusFinished,
			Result:    result,
		}
	}()

	return progressCh, nil
}

// ==================== 输出处理方法 ====================

// processArchitectOutput 处理 Architect Agent 的输出
func (w *RepoDocWorkflow) processArchitectOutput(content string) {
	klog.V(6).Infof("[RepoDocWorkflow] 处理 Architect 输出")

	// 尝试从输出中提取 JSON
	jsonStr := extractJSON(content)

	var result struct {
		RepoType  string            `json:"repo_type"`
		TechStack []string          `json:"tech_stack"`
		Summary   string            `json:"summary"`
		Chapters  []einodoc.Chapter `json:"chapters"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		klog.Warningf("[RepoDocWorkflow] 解析 Architect 输出失败: %v", err)
		return
	}

	// 更新状态
	w.state.SetRepoInfo(result.RepoType, result.TechStack)
	w.state.SetOutline(result.Chapters)

	klog.V(6).Infof("[RepoDocWorkflow] Architect 输出解析成功: type=%s, chapters=%d",
		result.RepoType, len(result.Chapters))
}

// processWriterOutput 处理 Writer Agent 的输出
func (w *RepoDocWorkflow) processWriterOutput(content string) {
	klog.V(6).Infof("[RepoDocWorkflow] 处理 Writer 输出")

	// Writer 的输出可能需要解析并保存到对应的小节
	// 这里简化处理，实际可能需要更复杂的解析逻辑
}

// processEditorOutput 处理 Editor Agent 的输出
func (w *RepoDocWorkflow) processEditorOutput(content string) {
	klog.V(6).Infof("[RepoDocWorkflow] 处理 Editor 输出")

	// 保存最终文档
	w.state.SetFinalDocument(content)
}

// buildResult 构建最终结果
func (w *RepoDocWorkflow) buildResult(localPath, finalContent string) *einodoc.RepoDocResult {
	// 如果状态中有大纲但没有小节内容，生成默认内容
	if len(w.state.Outline) > 0 && len(w.state.SectionsContent) == 0 {
		for chIdx, chapter := range w.state.Outline {
			for secIdx := range chapter.Sections {
				defaultContent := fmt.Sprintf("## %s\n\n%s 的内容待生成。\n\n",
					chapter.Sections[secIdx].Title, chapter.Title)
				w.state.SetSectionContent(chIdx, secIdx, defaultContent)
			}
		}
	}

	// 如果没有最终文档，生成一个
	if w.state.GetFinalDocument() == "" && finalContent != "" {
		w.state.SetFinalDocument(finalContent)
	}

	return &einodoc.RepoDocResult{
		LocalPath:       w.state.LocalPath,
		RepoType:        w.state.RepoType,
		TechStack:       w.state.TechStack,
		Outline:         w.state.Outline,
		Document:        w.state.GetFinalDocument(),
		SectionsCount:   len(w.state.SectionsContent),
		Completed:       true,
		SectionsContent: w.state.SectionsContent,
	}
}

// GetState 获取当前状态
func (w *RepoDocWorkflow) GetState() *einodoc.RepoDocState {
	return w.state
}

// ==================== Workflow 进度事件 ====================

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusError     WorkflowStatus = "error"
	WorkflowStatusFinished  WorkflowStatus = "finished"
)

// WorkflowProgressEvent 工作流进度事件
type WorkflowProgressEvent struct {
	Step      int                    `json:"step"`
	AgentName string                 `json:"agent_name"`
	Status    WorkflowStatus         `json:"status"`
	Content   string                 `json:"content"`
	Error     error                  `json:"error,omitempty"`
	Result    *einodoc.RepoDocResult `json:"result,omitempty"`
}

// ==================== 便捷函数 ====================

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}

// WorkflowInfo Workflow 信息
type WorkflowInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Agents      []string `json:"agents"`
}

// GetWorkflowInfo 获取 Workflow 信息
func (w *RepoDocWorkflow) GetWorkflowInfo() *WorkflowInfo {
	return &WorkflowInfo{
		Name:        "RepoDocWorkflow",
		Description: "基于 Eino ADK SequentialAgent 的仓库文档生成工作流",
		Agents: []string{
			AgentRepoInitializer,
			AgentArchitect,
			AgentExplorer,
			AgentWriter,
			AgentEditor,
		},
	}
}
