package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	etools "github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	"k8s.io/klog/v2"
)

// AgentFactory 负责创建各种子 Agent
// 使用 Eino ADK 原生的 ChatModelAgent
type AgentFactory struct {
	chatModel model.ToolCallingChatModel
	basePath  string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(chatModel model.ToolCallingChatModel, basePath string) *AgentFactory {
	return &AgentFactory{
		chatModel: chatModel,
		basePath:  basePath,
	}
}

// CreateRepoInitializerAgent 创建仓库初始化 Agent
// 负责克隆仓库和获取目录结构
func (f *AgentFactory) CreateRepoInitializerAgent() (adk.Agent, error) {
	role := AgentRoles[AgentRepoInitializer]

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        role.Name,
		Description: role.Description,
		Instruction: role.Instruction + `

你的任务是：
1. 使用 git_clone 工具克隆指定的代码仓库
2. 使用 list_dir 工具读取仓库的目录结构
3. 返回仓库的完整信息，包括：
   - 仓库 URL
   - 本地路径
   - 目录结构概要


请确保：
- 仓库成功克隆
- 获取完整的目录结构
- 返回的信息准确完整`,
		Model: f.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewGitCloneTool(f.basePath),
					tools.NewListDirTool(f.basePath),
				},
			},
		},
		MaxIterations: 10,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s agent: %w", role.Name, err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateArchitectAgent 创建架构师 Agent
// 负责分析仓库类型并生成文档大纲
func (f *AgentFactory) CreateArchitectAgent() (adk.Agent, error) {
	role := AgentRoles[AgentArchitect]

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        role.Name,
		Description: role.Description,
		Instruction: role.Instruction + `

你的任务是分析仓库并生成文档大纲：
1. 分析仓库的目录结构
2. 识别仓库类型（go/java/python/frontend/mixed）
3. 识别主要技术栈
4. 生成 2-3 个章节的文档大纲

输出格式必须是 JSON：
{
  "repo_type": "go",
  "tech_stack": ["Go", "Gin", "GORM"],
  "summary": "项目简介",
  "chapters": [
    {
      "title": "章节标题",
      "sections": [
        {"title": "小节标题", "hints": ["提示1", "提示2"]}
      ]
    }
  ]
}

请确保输出格式正确，可以被 JSON 解析。`,
		Model: f.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewSearchFilesTool(f.basePath),
					tools.NewListDirTool(f.basePath),
					tools.NewReadFileTool(f.basePath),
				},
			},
		},
		MaxIterations: 5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s agent: %w", role.Name, err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateExplorerAgent 创建探索者 Agent
// 负责深度分析代码结构
func (f *AgentFactory) CreateExplorerAgent() (adk.Agent, error) {
	role := AgentRoles[AgentExplorer]

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        role.Name,
		Description: role.Description,
		Instruction: role.Instruction + `

你的任务是深入探索代码库：
1. 读取 README 和关键配置文件（go.mod, package.json 等）
2. 搜索核心代码文件
3. 分析项目的主要模块和组件
4. 识别关键的函数、类和接口

请使用 read_file 和 search_files 工具来获取代码信息。`,
		Model: f.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewSearchFilesTool(f.basePath),
					tools.NewListDirTool(f.basePath),
					tools.NewReadFileTool(f.basePath),
				},
			},
		},
		MaxIterations: 15,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s agent: %w", role.Name, err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateWriterAgent 创建作者 Agent
// 负责生成文档内容
func (f *AgentFactory) CreateWriterAgent() (adk.Agent, error) {
	role := AgentRoles[AgentWriter]

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        role.Name,
		Description: role.Description,
		Instruction: role.Instruction + `

你的任务是为文档大纲的每个小节生成内容：
1. 根据章节和小节标题，撰写技术文档
2. 内容应包含：概念说明、代码示例、使用场景
3. 使用 Markdown 格式
4. 确保内容准确、清晰、专业

你可以使用 read_file 工具读取代码文件作为参考。
请为每个小节生成完整、独立的内容。`,
		Model: f.chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewReadFileTool(f.basePath),
				},
			},
		},
		MaxIterations: 20,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s agent: %w", role.Name, err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateEditorAgent 创建编辑 Agent
// 负责组装最终文档
func (f *AgentFactory) CreateEditorAgent() (adk.Agent, error) {
	role := AgentRoles[AgentEditor]

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        role.Name,
		Description: role.Description,
		Instruction: role.Instruction + `

你的任务是组装和优化最终文档：
1. 整合所有章节和小节的内容
2. 优化文档结构和格式
3. 添加文档头部信息（标题、仓库信息、技术栈）
4. 确保 Markdown 格式规范
5. 添加目录和导航链接

输出要求：
- 完整的 Markdown 文档
- 格式规范
- 结构清晰
- 可直接发布`,
		Model:         f.chatModel,
		MaxIterations: 5,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s agent: %w", role.Name, err)
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateSequentialAgent 创建顺序执行的 SequentialAgent
// 将所有子 Agent 按顺序组合
func (f *AgentFactory) CreateSequentialAgent() (adk.ResumableAgent, error) {
	ctx := context.Background()

	// 创建各个子 Agent
	initializer, err := f.CreateRepoInitializerAgent()
	if err != nil {
		return nil, err
	}

	architect, err := f.CreateArchitectAgent()
	if err != nil {
		return nil, err
	}

	explorer, err := f.CreateExplorerAgent()
	if err != nil {
		return nil, err
	}

	writer, err := f.CreateWriterAgent()
	if err != nil {
		return nil, err
	}

	editor, err := f.CreateEditorAgent()
	if err != nil {
		return nil, err
	}

	// 创建 SequentialAgent
	config := &adk.SequentialAgentConfig{
		Name:        "RepoDocSequentialAgent",
		Description: "仓库文档生成顺序执行 Agent - 按顺序执行初始化、分析、探索、撰写、编辑",
		SubAgents: []adk.Agent{
			initializer,
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
	return filepath.Join(basePath, etools.GenerateRepoDirName(repoURL))
}
