package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/service/einodoc"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
	"k8s.io/klog/v2"
)

// AgentFactory 负责创建各种子 Agent
// 每个 Agent 都是一个 ChatModelAgent，具有特定的角色和职责
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
func (f *AgentFactory) CreateRepoInitializerAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
	role := AgentRoles[AgentRepoInitializer]

	agent := &ChatModelAgentWrapper{
		name:        role.Name,
		description: role.Description,
		state:       state,
		basePath:    f.basePath,
		chatModel:   f.chatModel,
		doExecute:   f.executeRepoInitializer,
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateArchitectAgent 创建架构师 Agent
// 负责分析仓库结构并生成文档大纲
func (f *AgentFactory) CreateArchitectAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
	role := AgentRoles[AgentArchitect]

	agent := &ChatModelAgentWrapper{
		name:        role.Name,
		description: role.Description,
		state:       state,
		basePath:    f.basePath,
		chatModel:   f.chatModel,
		doExecute:   f.executeArchitect,
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateExplorerAgent 创建探索者 Agent
// 负责深度分析代码结构
func (f *AgentFactory) CreateExplorerAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
	role := AgentRoles[AgentExplorer]

	agent := &ChatModelAgentWrapper{
		name:        role.Name,
		description: role.Description,
		state:       state,
		basePath:    f.basePath,
		chatModel:   f.chatModel,
		doExecute:   f.executeExplorer,
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateWriterAgent 创建作者 Agent
// 负责生成文档内容
func (f *AgentFactory) CreateWriterAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
	role := AgentRoles[AgentWriter]

	agent := &ChatModelAgentWrapper{
		name:        role.Name,
		description: role.Description,
		state:       state,
		basePath:    f.basePath,
		chatModel:   f.chatModel,
		doExecute:   f.executeWriter,
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// CreateEditorAgent 创建编辑 Agent
// 负责组装最终文档
func (f *AgentFactory) CreateEditorAgent(state *StateManager) (*ChatModelAgentWrapper, error) {
	role := AgentRoles[AgentEditor]

	agent := &ChatModelAgentWrapper{
		name:        role.Name,
		description: role.Description,
		state:       state,
		basePath:    f.basePath,
		chatModel:   f.chatModel,
		doExecute:   f.executeEditor,
	}

	klog.V(6).Infof("[AgentFactory] 创建 %s Agent 成功", role.Name)
	return agent, nil
}

// ==================== Agent 执行逻辑 ====================

// executeRepoInitializer 执行仓库初始化
func (f *AgentFactory) executeRepoInitializer(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
	klog.V(6).Infof("[%s] 开始执行", AgentRepoInitializer)

	// 解析输入
	var args struct {
		RepoURL string `json:"repo_url"`
	}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		// 如果解析失败，尝试直接使用输入作为 URL
		args.RepoURL = input
	}

	if args.RepoURL == "" {
		return nil, fmt.Errorf("repo_url is required")
	}

	// 步骤1: 克隆仓库
	cloneTool := tools.NewGitCloneTool(f.basePath)
	cloneArgs, _ := json.Marshal(map[string]string{
		"repo_url":   args.RepoURL,
		"target_dir": tools.GenerateRepoDirName(args.RepoURL),
	})

	klog.V(6).Infof("[%s] 克隆仓库: %s", AgentRepoInitializer, args.RepoURL)
	cloneResult, err := cloneTool.InvokableRun(ctx, string(cloneArgs))
	if err != nil {
		klog.Errorf("[%s] 克隆失败: %v", AgentRepoInitializer, err)
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	// 设置本地路径
	localPath := filepath.Join(f.basePath, tools.GenerateRepoDirName(args.RepoURL))
	state.SetLocalPath(localPath)

	// 步骤2: 获取目录结构
	listTool := tools.NewListDirTool(f.basePath)
	listArgs, _ := json.Marshal(map[string]interface{}{
		"dir":       tools.GenerateRepoDirName(args.RepoURL),
		"recursive": true,
	})

	klog.V(6).Infof("[%s] 获取目录结构", AgentRepoInitializer)
	treeResult, err := listTool.InvokableRun(ctx, string(listArgs))
	if err != nil {
		klog.Errorf("[%s] 获取目录结构失败: %v", AgentRepoInitializer, err)
		return nil, fmt.Errorf("list dir failed: %w", err)
	}

	state.SetRepoTree(treeResult)

	// 返回结果
	result := fmt.Sprintf("仓库初始化完成！\n\n克隆结果: %s\n\n目录结构:\n%s", cloneResult, treeResult)
	klog.V(6).Infof("[%s] 执行完成", AgentRepoInitializer)

	return &schema.Message{
		Role:    schema.Assistant,
		Content: result,
	}, nil
}

// executeArchitect 执行架构分析
func (f *AgentFactory) executeArchitect(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
	klog.V(6).Infof("[%s] 开始执行", AgentArchitect)

	repoTree := state.GetRepoTree()
	if repoTree == "" {
		return nil, fmt.Errorf("repo tree is empty, please run RepoInitializer first")
	}

	// 使用 LLM 分析仓库类型并生成大纲
	messages := []*schema.Message{
		{
			Role: schema.System,
			Content: `你是仓库分析专家。请分析仓库目录结构并提供：
1) 仓库类型（go/java/python/frontend/mixed）
2) 技术栈
3) 简要总结
4) 文档大纲（2-3个章节，每个章节2-3个小节）

请按照下面的 JSON 格式回复：
{
  "repo_type": "go",
  "tech_stack": ["Go", "Gin", "GORM"],
  "summary": "这是一个基于 Go 的 Web 服务项目",
  "chapters": [
    {
      "title": "项目概述",
      "sections": [
        {"title": "项目简介", "hints": ["项目背景", "核心功能"]},
        {"title": "技术架构", "hints": ["架构设计", "技术选型"]}
      ]
    }
  ]
}`,
		},
		{
			Role:    schema.User,
			Content: fmt.Sprintf("请分析以下仓库结构:\n\n%s", repoTree),
		},
	}

	klog.V(6).Infof("[%s] 调用 LLM 分析仓库", AgentArchitect)
	resp, err := f.chatModel.Generate(ctx, messages)
	if err != nil {
		klog.Warningf("[%s] LLM 分析失败，使用默认值: %v", AgentArchitect, err)
		// 使用默认值
		state.SetRepoInfo("unknown", []string{})
		state.SetOutline([]einodoc.Chapter{
			{
				Title: "项目概述",
				Sections: []einodoc.Section{
					{Title: "项目简介", Hints: []string{"项目概述"}},
					{Title: "系统架构", Hints: []string{"系统架构"}},
				},
			},
		})
	} else {
		// 解析 JSON 响应
		content := extractJSON(resp.Content)
		var result struct {
			RepoType  string            `json:"repo_type"`
			TechStack []string          `json:"tech_stack"`
			Summary   string            `json:"summary"`
			Chapters  []einodoc.Chapter `json:"chapters"`
		}

		if err := json.Unmarshal([]byte(content), &result); err != nil {
			klog.Warningf("[%s] JSON 解析失败，使用默认值: %v", AgentArchitect, err)
			state.SetRepoInfo("unknown", []string{})
			state.SetOutline([]einodoc.Chapter{
				{
					Title: "项目概述",
					Sections: []einodoc.Section{
						{Title: "项目简介", Hints: []string{"项目概述"}},
					},
				},
			})
		} else {
			state.SetRepoInfo(result.RepoType, result.TechStack)
			state.SetOutline(result.Chapters)
			klog.V(6).Infof("[%s] 分析完成: type=%s, chapters=%d", AgentArchitect, result.RepoType, len(result.Chapters))
		}
	}

	repoType, techStack := state.GetRepoInfo()
	outline := state.GetOutline()

	result := fmt.Sprintf("仓库分析完成！\n\n类型: %s\n技术栈: %v\n大纲章节数: %d", repoType, techStack, len(outline))
	klog.V(6).Infof("[%s] 执行完成", AgentArchitect)

	return &schema.Message{
		Role:    schema.Assistant,
		Content: result,
	}, nil
}

// executeExplorer 执行代码探索
func (f *AgentFactory) executeExplorer(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
	klog.V(6).Infof("[%s] 开始执行", AgentExplorer)

	// 获取关键文件信息
	localPath := state.GetLocalPath()
	if localPath == "" {
		return nil, fmt.Errorf("local path is empty")
	}

	// 读取 README 文件（如果存在）
	readmeTool := tools.NewReadFileTool(f.basePath)
	readmeArgs, _ := json.Marshal(map[string]string{
		"path": filepath.Join(tools.GenerateRepoDirName(state.GetState().RepoURL), "README.md"),
	})

	readmeContent, _ := readmeTool.InvokableRun(ctx, string(readmeArgs))
	if readmeContent == "" {
		readmeArgs, _ = json.Marshal(map[string]string{
			"path": filepath.Join(tools.GenerateRepoDirName(state.GetState().RepoURL), "README"),
		})
		readmeContent, _ = readmeTool.InvokableRun(ctx, string(readmeArgs))
	}

	// 根据仓库类型读取关键配置文件
	repoType, _ := state.GetRepoInfo()
	var configContent string

	switch repoType {
	case "go":
		goModArgs, _ := json.Marshal(map[string]string{
			"path": filepath.Join(tools.GenerateRepoDirName(state.GetState().RepoURL), "go.mod"),
		})
		configContent, _ = readmeTool.InvokableRun(ctx, string(goModArgs))
	case "python":
		reqArgs, _ := json.Marshal(map[string]string{
			"path": filepath.Join(tools.GenerateRepoDirName(state.GetState().RepoURL), "requirements.txt"),
		})
		configContent, _ = readmeTool.InvokableRun(ctx, string(reqArgs))
	case "frontend", "node", "javascript", "typescript":
		pkgArgs, _ := json.Marshal(map[string]string{
			"path": filepath.Join(tools.GenerateRepoDirName(state.GetState().RepoURL), "package.json"),
		})
		configContent, _ = readmeTool.InvokableRun(ctx, string(pkgArgs))
	}

	// 记录探索结果
	exploreResult := fmt.Sprintf("代码探索完成！\n\nREADME 内容:\n%s\n\n配置文件:\n%s",
		truncate(readmeContent, 1000),
		truncate(configContent, 500))

	klog.V(6).Infof("[%s] 执行完成", AgentExplorer)

	return &schema.Message{
		Role:    schema.Assistant,
		Content: exploreResult,
	}, nil
}

// executeWriter 执行文档撰写
func (f *AgentFactory) executeWriter(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
	klog.V(6).Infof("[%s] 开始执行", AgentWriter)

	outline := state.GetOutline()
	if len(outline) == 0 {
		return nil, fmt.Errorf("outline is empty")
	}

	repoType, techStack := state.GetRepoInfo()
	totalSections := 0
	generatedSections := 0

	// 为每个小节生成内容
	for chIdx, chapter := range outline {
		for secIdx, section := range chapter.Sections {
			totalSections++

			// 检查是否已有内容
			existing := state.GetSectionContent(chIdx, secIdx)
			if existing != "" {
				klog.V(6).Infof("[%s] 章节[%d]小节[%d]已有内容，跳过", AgentWriter, chIdx, secIdx)
				generatedSections++
				continue
			}

			// 使用 LLM 生成内容
			messages := []*schema.Message{
				{
					Role:    schema.System,
					Content: fmt.Sprintf("你是技术文档撰写者。请为 %s 项目撰写技术文档。技术栈: %v", repoType, techStack),
				},
				{
					Role: schema.User,
					Content: fmt.Sprintf(`请撰写文档内容：
章节: %s
小节: %s
写作要点: %v

请以 Markdown 格式撰写，内容应准确、清晰、专业。`,
						chapter.Title, section.Title, section.Hints),
				},
			}

			resp, err := f.chatModel.Generate(ctx, messages)
			if err != nil {
				klog.Warningf("[%s] 生成内容失败: %v", AgentWriter, err)
				// 使用默认内容
				defaultContent := fmt.Sprintf("## %s\n\n%s 章节下的 %s 小节内容。\n\n*此内容由 ADK WriterAgent 生成*",
					section.Title, chapter.Title, section.Title)
				state.SetSectionContent(chIdx, secIdx, defaultContent)
			} else {
				state.SetSectionContent(chIdx, secIdx, resp.Content)
				generatedSections++
			}
		}
	}

	result := fmt.Sprintf("文档撰写完成！\n\n总小节数: %d\n已生成: %d", totalSections, generatedSections)
	klog.V(6).Infof("[%s] 执行完成: %s", AgentWriter, result)

	return &schema.Message{
		Role:    schema.Assistant,
		Content: result,
	}, nil
}

// executeEditor 执行文档编辑
func (f *AgentFactory) executeEditor(ctx context.Context, state *StateManager, input string) (*schema.Message, error) {
	klog.V(6).Infof("[%s] 开始执行", AgentEditor)

	// 构建最终结果
	result := state.BuildResult()

	summary := fmt.Sprintf(`文档编辑完成！

仓库: %s
类型: %s
技术栈: %v
章节数: %d
小节数: %d
文档长度: %d 字符

文档已成功生成！`,
		result.RepoURL,
		result.RepoType,
		result.TechStack,
		len(result.Outline),
		result.SectionsCount,
		len(result.Document),
	)

	klog.V(6).Infof("[%s] 执行完成", AgentEditor)

	return &schema.Message{
		Role:    schema.Assistant,
		Content: summary,
	}, nil
}

// ==================== 辅助函数 ====================

// extractJSON 从文本中提取 JSON 部分
func extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start >= 0 && end > start {
		return content[start : end+1]
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
