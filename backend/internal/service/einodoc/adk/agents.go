package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	"k8s.io/klog/v2"
)

// AgentName 定义各个子 Agent 的名称常量
const (
	// AgentRepoInitializer 仓库初始化 Agent - 负责克隆仓库和基础分析
	AgentRepoInitializer = "RepoInitializer"
	// AgentArchitect 架构师 Agent - 负责生成文档大纲
	AgentArchitect = "Architect"
	// AgentExplorer 探索者 Agent - 负责深度代码分析
	AgentExplorer = "Explorer"
	// AgentWriter 作者 Agent - 负责生成文档内容
	AgentWriter = "Writer"
	// AgentEditor 编辑 Agent - 负责组装最终文档
	AgentEditor = "Editor"
)

// modelProvider 实现 adkagents.ModelProvider
type modelProvider struct {
	chatModel model.ToolCallingChatModel
}

// GetModel 获取指定名称的模型，name 为空时返回默认模型
func (p *modelProvider) GetModel(name string) (model.ToolCallingChatModel, error) {
	// 目前只支持默认模型
	return p.chatModel, nil
}

// DefaultModel 获取默认模型
func (p *modelProvider) DefaultModel() model.ToolCallingChatModel {
	return p.chatModel
}

// toolProvider 实现 adkagents.ToolProvider
type toolProvider struct {
	basePath string
}

// GetTool 获取指定名称的工具
func (p *toolProvider) GetTool(name string) (tool.BaseTool, error) {
	switch name {
	case "list_dir":
		return tools.NewListDirTool(p.basePath), nil
	case "read_file":
		return tools.NewReadFileTool(p.basePath), nil
	case "search_files":
		return tools.NewSearchFilesTool(p.basePath), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ListTools 列出所有可用工具名称
func (p *toolProvider) ListTools() []string {
	return []string{"list_dir", "read_file", "search_files"}
}

// AgentFactory 负责创建各种子 Agent
// 使用 adkagents.Manager 管理基础 Agent 的加载和创建
type AgentFactory struct {
	manager  *adkagents.Manager
	basePath string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(chatModel model.ToolCallingChatModel, basePath string) (*AgentFactory, error) {
	// 创建 providers
	mp := &modelProvider{chatModel: chatModel}
	tp := &toolProvider{basePath: basePath}

	// 创建 Manager
	config := &adkagents.Config{
		Dir:            "../agents",
		AutoReload:     true,
		ReloadInterval: 5 * time.Second,
		ModelProvider:  mp,
		ToolProvider:   tp,
	}

	manager, err := adkagents.NewManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adkagents manager: %w", err)
	}

	return &AgentFactory{
		manager:  manager,
		basePath: basePath,
	}, nil
}

// GetAgent 获取指定名称的基础 Agent
// 这是获取基础 Agent 的推荐方式
func (f *AgentFactory) GetAgent(name string) (adk.Agent, error) {
	return f.manager.GetAgent(name)
}

// CreateSequentialAgent 创建顺序执行的 SequentialAgent
// 将所有子 Agent 按顺序组合
// 注意：此方法保持既有逻辑，不由 adkagents.Manager 直接管理
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
	ctx := context.Background()

	architect, err := f.manager.GetAgent(AgentArchitect)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentArchitect, err)
	}

	explorer, err := f.manager.GetAgent(AgentExplorer)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentExplorer, err)
	}

	writer, err := f.manager.GetAgent(AgentWriter)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentWriter, err)
	}

	editor, err := f.manager.GetAgent(AgentEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s agent: %w", AgentEditor, err)
	}

	// 创建 SequentialAgent
	config := &adk.SequentialAgentConfig{
		Name:        "RepoDocSequentialAgent",
		Description: "仓库文档生成顺序执行 Agent - 按顺序执行初始化、分析、探索、撰写、编辑",
		SubAgents: []adk.Agent{
			architect,
			explorer,
			writer,
			editor,
		},
	}

	sequentialAgent, err := adk.NewSequentialAgent(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sequential agent: %w", err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 SequentialAgent 成功")
	return sequentialAgent, nil
}

// Stop 停止 AgentFactory，释放资源
func (f *AgentFactory) Stop() {
	if f.manager != nil {
		f.manager.Stop()
	}
}

// ==================== Workflow 辅助函数 ====================

// BuildWorkflowInput 构建 Workflow 输入
func BuildWorkflowInput(repoURL string) *adk.AgentInput {
	return &adk.AgentInput{
		Messages: []adk.Message{
			{
				Role:    schema.User,
				Content: fmt.Sprintf(`{"repo_url": "%s"}`, repoURL),
			},
		},
	}
}

// ParseAgentEvent 解析 Agent 事件，提取文本内容
func ParseAgentEvent(event *adk.AgentEvent) string {
	if event == nil {
		return ""
	}

	if event.Err != nil {
		return fmt.Sprintf("Error: %v", event.Err)
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		return event.Output.MessageOutput.Message.Content
	}

	return ""
}

// ExtractRepoInfoFromContent 从 Agent 输出内容提取仓库信息
func ExtractRepoInfoFromContent(content string) (*einodoc.RepoDocState, error) {
	// 尝试解析 JSON
	var result struct {
		RepoType  string            `json:"repo_type"`
		TechStack []string          `json:"tech_stack"`
		Summary   string            `json:"summary"`
		Chapters  []einodoc.Chapter `json:"chapters"`
		LocalPath string            `json:"local_path"`
	}

	if err := json.Unmarshal([]byte(extractJSON(content)), &result); err != nil {
		// 如果不是 JSON，返回空状态
		return nil, fmt.Errorf("failed to parse repo info: %w", err)
	}

	state := einodoc.NewRepoDocState("", result.LocalPath)
	state.SetRepoInfo(result.RepoType, result.TechStack)
	state.SetOutline(result.Chapters)

	return state, nil
}

// extractJSON 从文本中提取 JSON 部分
func extractJSON(content string) string {
	start := -1
	end := -1
	depth := 0

	for i, ch := range content {
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}

	if start >= 0 && end > start {
		return content[start:end]
	}

	return content
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetLocalPathFromRepoURL 根据仓库 URL 获取本地路径
func GetLocalPathFromRepoURL(basePath, repoURL string) string {
	return filepath.Join(basePath, tools.GenerateRepoDirName(repoURL))
}
