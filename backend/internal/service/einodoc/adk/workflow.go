package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	factory, err := NewAgentFactory(chatModel, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent factory: %w", err)
	}
	return &RepoDocWorkflow{
		basePath:  basePath,
		chatModel: chatModel,
		factory:   factory,
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

	// 循环处理每个 agent 的执行结果
	for {
		select {
		case <-ctx.Done():
			// 检查上下文是否被取消，避免长时间挂起
			klog.Warningf("[RepoDocWorkflow.Run] 上下文被取消: %v", ctx.Err())
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
			// 继续正常执行
		}

		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			// 检查是否是迭代次数超限的错误
			errMsg := event.Err.Error()
			if strings.Contains(errMsg, "exceeds max iterations") || strings.Contains(errMsg, "max iterations") {
				klog.Warningf("[RepoDocWorkflow.Run] 检测到迭代次数超限错误，执行Editor Agent进行最终总结: %v", event.Err)

				// 创建Editor Agent来生成最终文档
				editorAgent, err := w.factory.GetAgent(AgentEditor)
				if err != nil {
					klog.Errorf("[RepoDocWorkflow.Run] 创建Editor Agent失败: %v", err)
					return nil, fmt.Errorf("failed to create editor agent: %w", err)
				}

				// 准备最终的消息内容，包含当前已有信息
				summaryMsg := `你是一个文档编辑助手。基于之前收集到的信息，生成一份完整的项目文档。

现有信息摘要：
- 已探索的章节：` + fmt.Sprintf("%d", len(w.state.Outline)) + `
- 已生成的小节数：` + fmt.Sprintf("%d", len(w.state.SectionsContent)) + `
- 当前状态：分析过程因达到最大迭代次数而终止

请根据现有信息和通用知识，生成最终的技术文档。`

				// 运行Editor Agent进行总结
				runnerForEditor := adk.NewRunner(ctx, adk.RunnerConfig{
					Agent: editorAgent,
				})

				editorIter := runnerForEditor.Run(ctx, []adk.Message{
					{
						Role:    schema.User,
						Content: summaryMsg,
					},
				})

				// 获取Editor的输出
				for {
					editorEvent, editorOk := editorIter.Next()
					if !editorOk {
						break
					}

					if editorEvent.Err != nil {
						klog.Warningf("[RepoDocWorkflow.Run] Editor Agent执行时出错: %v", editorEvent.Err)
						break
					}

					if editorEvent.Output != nil && editorEvent.Output.MessageOutput != nil {
						content := editorEvent.Output.MessageOutput.Message.Content
						w.processEditorOutput(content)
						lastContent = content
						klog.V(6).Infof("[RepoDocWorkflow.Run] Editor Agent生成最终内容，长度: %d", len(content))

						if editorEvent.Action != nil && editorEvent.Action.Exit {
							break
						}
					}
				}

				// 跳出主循环，使用已收集的内容构建结果
				break
			} else {
				klog.Errorf("[RepoDocWorkflow.Run] Agent 执行出错: %v", event.Err)
				return nil, fmt.Errorf("agent execution failed: %w", event.Err)
			}
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
	klog.V(6).Infof("[RepoDocWorkflow] 处理 Writer 输出, 内容长度: %d", len(content))

	// Writer 的输出可能需要解析并保存到对应的小节
	// 这里简化处理，实际可能需要更复杂的解析逻辑
	if content != "" {
		// 如果有内容，则记录或处理它
		klog.V(6).Infof("[RepoDocWorkflow] Writer 输出内容预览: %.100s", content)
	}
}

// processEditorOutput 处理 Editor Agent 的输出
func (w *RepoDocWorkflow) processEditorOutput(content string) {
	klog.V(6).Infof("[RepoDocWorkflow] 处理 Editor 输出")

	// 保存最终文档
	w.state.SetFinalDocument(content)
}

// buildResult 构建最终结果
func (w *RepoDocWorkflow) buildResult(localPath, finalContent string) *einodoc.RepoDocResult {
	// 如果状态中的本地路径为空，使用传入的路径
	if w.state.LocalPath == "" {
		w.state.LocalPath = localPath
	}

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
func ToJSON(v any) string {
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
