package einodoc

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
)

// WorkflowInput Workflow 输入
// 作为 Chain 的输入类型
type WorkflowInput struct {
	RepoURL string `json:"repo_url"` // 仓库 Git URL
}

// WorkflowOutput Workflow 输出
// 作为 Chain 的输出类型
type WorkflowOutput struct {
	State  *RepoDocState  `json:"state"`  // Workflow 执行过程中的状态
	Result *RepoDocResult `json:"result"` // 最终结果
}

// RepoDocChain 基于 Chain 的简化 Workflow
// 使用 compose.Chain 编排一系列处理步骤
type RepoDocChain struct {
	chain *compose.Chain[WorkflowInput, WorkflowOutput]
}

// NewRepoDocChain 创建 RepoDoc Chain
// basePath: 仓库存储的基础路径
// chatModel: Eino ChatModel 实例，用于 LLM 调用
// 返回: 配置好的 Chain 实例或错误
func NewRepoDocChain(basePath string, chatModel model.ChatModel) (*RepoDocChain, error) {
	klog.V(6).Infof("[NewRepoDocChain] 开始创建 RepoDocChain: basePath=%s", basePath)

	chain := compose.NewChain[WorkflowInput, WorkflowOutput]()

	// ========== Step 1: Clone & Read Tree ==========
	// 克隆仓库并读取目录结构
	klog.V(6).Infof("[NewRepoDocChain] 添加 Step 1: Clone & Read Tree")
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input WorkflowInput) (WorkflowOutput, error) {
		klog.V(6).Infof("[Workflow Step 1] 开始执行: Clone & Read Tree")
		klog.V(6).Infof("[Workflow Step 1] 输入参数: repoURL=%s", input.RepoURL)

		state := NewRepoDocState(input.RepoURL, "")

		// 使用 git_clone 工具克隆仓库
		klog.V(6).Infof("[Workflow Step 1] 调用 GitCloneTool")
		cloneTool := tools.NewGitCloneTool(basePath)
		cloneArgs, _ := json.Marshal(map[string]string{
			"repo_url":   input.RepoURL,
			"target_dir": tools.GenerateRepoDirName(input.RepoURL),
		})
		klog.V(6).Infof("[Workflow Step 1] GitCloneTool 参数: %s", string(cloneArgs))

		cloneResult, err := cloneTool.InvokableRun(ctx, string(cloneArgs))
		if err != nil {
			klog.Errorf("[Workflow Step 1] GitCloneTool 执行失败: %v", err)
			return WorkflowOutput{}, fmt.Errorf("clone failed: %w", err)
		}
		klog.V(6).Infof("[Workflow Step 1] GitCloneTool 执行成功: 仓库路径=%s, 结果=%s", state.LocalPath, cloneResult)

		state.LocalPath = filepath.Join(basePath, tools.GenerateRepoDirName(input.RepoURL))
		klog.V(6).Infof("[Workflow Step 1] 设置本地路径: %s", state.LocalPath)

		// 读取目录结构
		klog.V(6).Infof("[Workflow Step 1] 调用 ListDirTool")
		listTool := tools.NewListDirTool(basePath)
		listArgs, _ := json.Marshal(map[string]interface{}{
			"dir":       tools.GenerateRepoDirName(input.RepoURL),
			"recursive": true,
		})

		treeResult, err := listTool.InvokableRun(ctx, string(listArgs))
		if err != nil {
			klog.Errorf("[Workflow Step 1] ListDirTool 执行失败: %v", err)
			return WorkflowOutput{}, fmt.Errorf("list dir failed: %w", err)
		}
		klog.V(6).Infof("[Workflow Step 1] ListDirTool 执行成功: 目录结构长度=%d", len(treeResult))

		// 将 treeResult 存储到 state 中供后续使用
		state.SetRepoTree(treeResult)

		klog.V(6).Infof("[Workflow Step 1] 执行完成")
		return WorkflowOutput{
			State: state,
			Result: &RepoDocResult{
				RepoURL:   input.RepoURL,
				LocalPath: state.LocalPath,
				Outline:   []Chapter{},
			},
		}, nil
	}))

	// ========== Step 2: Pre-read Analysis with LLM ==========
	// 使用 LLM 分析仓库类型和技术栈
	klog.V(6).Infof("[NewRepoDocChain] 添加 Step 2: Pre-read Analysis")
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		klog.V(6).Infof("[Workflow Step 2] 开始执行: Pre-read Analysis with LLM")

		state := output.State

		// 获取目录结构
		treeResult := state.RepoTree
		if treeResult == "" {
			klog.Warningf("[Workflow Step 2] State 中目录结构为空，尝试重新读取")
			listTool := tools.NewListDirTool(basePath)
			listArgs, _ := json.Marshal(map[string]interface{}{
				"dir":       tools.GenerateRepoDirName(state.RepoURL),
				"recursive": true,
			})
			var err error
			treeResult, err = listTool.InvokableRun(ctx, string(listArgs))
			if err != nil {
				klog.Warningf("[Workflow Step 2] ListDirTool 执行失败，使用默认值: %v", err)
				treeResult = "Failed to read directory"
			}
		}
		klog.V(6).Infof("[Workflow Step 2] 获取目录结构成功: 内容长度=%d", len(treeResult))

		// 使用 LLM 分析仓库
		klog.V(6).Infof("[Workflow Step 2] 调用 LLM 分析仓库")
		messages := []*schema.Message{
			{
				Role: schema.System,
				Content: `您是代码仓库分析器。
				请分析仓库结构并提供：
				1）仓库类型（go/java/python/frontend/mixed），
				2）技术栈，
				3）简要总结。
				请按照下面的 JSON 格式回复。
				{
  "repo_type": "go",
  "tech_stack": ["Go", "Docker", "Kubernetes"],
  "summary": "这是一个基于 Go 语言的微服务项目，使用 Docker 和 Kubernetes 进行部署。"
}
				`,
			},
			{
				Role:    schema.User,
				Content: fmt.Sprintf("仓库地址: %s\n\n目录结构:\n%s", state.RepoURL, treeResult),
			},
		}
		klog.V(6).Infof("[Workflow Step 2] LLM 请求: messageCount=%d", len(messages))

		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			klog.Warningf("[Workflow Step 2] LLM 分析失败，使用默认值: %v", err)
			state.SetRepoInfo("unknown", []string{})
		} else {
			// 解析 JSON 响应
			var analysis struct {
				RepoType  string   `json:"repo_type"`
				TechStack []string `json:"tech_stack"`
				Summary   string   `json:"summary"`
			}
			content := extractJSON(resp.Content)
			klog.V(6).Infof("[Workflow Step 2] LLM 响应解析: contentLength=%d", len(content))

			if err := json.Unmarshal([]byte(content), &analysis); err != nil {
				klog.Warningf("[Workflow Step 2] JSON 解析失败，使用默认值: %v", err)
				state.SetRepoInfo("unknown", []string{})
			} else {
				state.SetRepoInfo(analysis.RepoType, analysis.TechStack)
				klog.V(6).Infof("[Workflow Step 2] 分析成功: repoType=%s, techStack=%v",
					analysis.RepoType, analysis.TechStack)
			}
		}

		output.Result.RepoType = state.RepoType
		output.Result.TechStack = state.TechStack
		klog.V(6).Infof("[Workflow Step 2] 执行完成")

		return output, nil
	}))

	// ========== Step 3: Generate Outline with LLM ==========
	// 使用 LLM 生成文档大纲
	klog.V(6).Infof("[NewRepoDocChain] Adding Step 3: 生成文档大纲")
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		klog.V(6).Infof("[Workflow Step 3] 开始执行: 用大模型生成文档大纲")

		state := output.State
		klog.V(6).Infof("[Workflow Step 3] 当前状态: repoType=%s, techStack=%v", state.RepoType, state.TechStack)

		messages := []*schema.Message{
			{
				Role: schema.System,
				Content: `您是技术文档专家，请为该仓库创建文档大纲。
				每个章节应包含2-3个段落，每个段落应包含2-3个提示。
				请根据仓库类型和技术栈，生成一个2-3章节的文档大纲。

Respond in JSON format:
{
  "chapters": [
    {
      "title": "章节标题",
      "sections": [
        {"title": "段落标题", "hints": ["提示1", "提示2"]}
      ]
    }
  ]
}`,
			},
			{
				Role: schema.User,
				Content: fmt.Sprintf("仓库类型: %s\n技术栈: %v\n\n请根据仓库类型和技术栈，生成一个2-3章节的文档大纲。",
					state.RepoType, state.TechStack),
			},
		}
		klog.V(6).Infof("[Workflow Step 3] LLM 请求: messageCount=%d", len(messages))

		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			klog.Warningf("[Workflow Step 3] LLM 生成大纲失败，使用默认大纲: %v", err)
			state.SetOutline([]Chapter{
				{
					Title: "项目总览",
					Sections: []Section{
						{Title: "项目介绍", Hints: []string{"项目总览"}},
						{Title: "系统架构", Hints: []string{"系统架构"}},
					},
				},
			})
		} else {
			var outline struct {
				Chapters []Chapter `json:"chapters"`
			}
			content := extractJSON(resp.Content)
			klog.V(6).Infof("[Workflow Step 3] LLM 响应解析: contentLength=%d", len(content))

			if err := json.Unmarshal([]byte(content), &outline); err != nil {
				klog.Warningf("[Workflow Step 3] JSON 解析失败，使用默认大纲: %v", err)
				state.SetOutline([]Chapter{
					{
						Title: "项目概述",
						Sections: []Section{
							{Title: "项目介绍", Hints: []string{"项目概述"}},
						},
					},
				})
			} else {
				state.SetOutline(outline.Chapters)
				klog.V(6).Infof("[Workflow Step 3] 大纲生成成功: 章节数=%d, 段落数=%d",
					len(outline.Chapters), len(outline.Chapters))
			}
		}

		output.Result.Outline = state.Outline
		klog.V(6).Infof("[Workflow Step 3] 执行完成")
		return output, nil
	}))

	// ========== Step 4: Generate Section Content (Simplified) ==========
	// 为每个 section 生成内容
	klog.V(6).Infof("[NewRepoDocChain] 添加 Step 4: Generate Section Content")
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		klog.V(6).Infof("[Workflow Step 4] 开始执行: Generate Section Content")

		state := output.State
		klog.V(6).Infof("[Workflow Step 4] 当前大纲: chapters=%d", len(state.Outline))

		// 为每个 section 生成简单的内容
		sectionCount := 0
		for chIdx, chapter := range state.Outline {
			klog.V(6).Infof("[Workflow Step 4] 处理 Chapter[%d]: %s, sections=%d", chIdx, chapter.Title, len(chapter.Sections))

			for secIdx, section := range chapter.Sections {
				_ = sectionKey(chIdx, secIdx) // 避免未使用变量
				sectionCount++

				klog.V(6).Infof("[Workflow Step 4]   生成 Section[%d/%d]: %s", chIdx, secIdx, section.Title)

				// 使用 LLM 生成内容
				messages := []*schema.Message{
					{
						Role:    schema.System,
						Content: "您是技术文档撰写者，请以 Markdown 格式撰写一段简短的文档章节。",
					},
					{
						Role: schema.User,
						Content: fmt.Sprintf("章节: %s\n段落: %s\n提示: %v\n\n请撰写一个简短的段落，介绍这个主题。",
							chapter.Title, section.Title, section.Hints),
					},
				}

				resp, err := chatModel.Generate(ctx, messages)
				if err != nil {
					klog.Warningf("[Workflow Step 4]   Section 内容生成失败，使用默认内容: %v", err)
					state.SetSectionContent(chIdx, secIdx, fmt.Sprintf("## %s\n\n%s 下 %s 的内容。\n\n*由 Eino Workflow 生成*",
						section.Title, chapter.Title, section.Title))
				} else {
					state.SetSectionContent(chIdx, secIdx, resp.Content)
					klog.V(6).Infof("[Workflow Step 4]   Section 内容生成成功: length=%d", len(resp.Content))
				}
			}
		}

		klog.V(6).Infof("[Workflow Step 4] 执行完成: totalSections=%d", sectionCount)
		return output, nil
	}))

	// ========== Step 5: Finalize Document ==========
	// 组装最终文档
	klog.V(6).Infof("[NewRepoDocChain] 添加 Step 5: Finalize Document")
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		klog.V(6).Infof("[Workflow Step 5] 开始执行: Finalize Document")

		state := output.State
		klog.V(6).Infof("[Workflow Step 5] 当前状态: %s", utils.ToJSON(state))
		// 组装最终文档
		klog.V(6).Infof("[Workflow Step 5] 组装文档头部信息")
		var doc strings.Builder
		doc.WriteString(fmt.Sprintf("# 项目文档\n\n"))
		doc.WriteString(fmt.Sprintf("**仓库:** %s\n\n", state.RepoURL))
		doc.WriteString(fmt.Sprintf("**类型:** %s\n\n", state.RepoType))
		doc.WriteString(fmt.Sprintf("**技术栈:** %v\n\n", state.TechStack))
		doc.WriteString("---\n\n")

		klog.V(6).Infof("[Workflow Step 5] 组装章节内容: chapters=%d", len(state.Outline))
		for chIdx, chapter := range state.Outline {
			doc.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))
			klog.V(6).Infof("[Workflow Step 5]   章节[%d]: %s, 段落=%d", chIdx, chapter.Title, len(chapter.Sections))

			for secIdx, section := range chapter.Sections {
				doc.WriteString(fmt.Sprintf("### %s\n\n", section.Title))

				content := state.GetSectionContent(chIdx, secIdx)
				if content != "" {
					doc.WriteString(content)
					doc.WriteString("\n\n")
				}
			}
		}

		finalDoc := doc.String()
		state.SetFinalDocument(finalDoc)
		output.Result.Document = finalDoc
		output.Result.SectionsContent = state.SectionsContent
		output.Result.SectionsCount = len(state.SectionsContent)
		output.Result.Completed = true

		klog.V(6).Infof("[Workflow Step 5] 文档组装完成: length=%d, sections=%d", len(finalDoc), output.Result.SectionsCount)
		klog.V(6).Infof("[Workflow] 所有步骤执行完成")

		return output, nil
	}))

	klog.V(6).Infof("[NewRepoDocChain] RepoDocChain 创建完成")
	return &RepoDocChain{chain: chain}, nil
}

// Run 执行 Chain
// 编译并执行 Chain，生成最终文档
// ctx: 上下文
// input: Workflow 输入
// 返回: RepoDocResult 或错误
func (c *RepoDocChain) Run(ctx context.Context, input WorkflowInput) (*RepoDocResult, error) {
	klog.V(6).Infof("[RepoDocChain.Run] 开始执行 Chain: repoURL=%s", input.RepoURL)

	klog.V(6).Infof("[RepoDocChain.Run] 编译 Chain")
	runnable, err := c.chain.Compile(ctx)
	if err != nil {
		klog.Errorf("[RepoDocChain.Run] Chain 编译失败: %v", err)
		return nil, fmt.Errorf("failed to compile chain: %w", err)
	}
	klog.V(6).Infof("[RepoDocChain.Run] Chain 编译成功")

	klog.V(6).Infof("[RepoDocChain.Run] 调用 Chain.Invoke")
	output, err := runnable.Invoke(ctx, input)
	if err != nil {
		klog.Errorf("[RepoDocChain.Run] Chain 执行失败: %v", err)
		return nil, fmt.Errorf("chain execution failed: %w", err)
	}

	klog.V(6).Infof("[RepoDocChain.Run] Chain 执行成功: documentLength=%d, sections=%d",
		len(output.Result.Document), output.Result.SectionsCount)

	return output.Result, nil
}

// extractJSON 从文本中提取 JSON 部分
// 查找第一个 { 和最后一个 } 之间的内容
// content: 可能包含 JSON 的文本
// 返回: 提取的 JSON 字符串
func extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start >= 0 && end > start {
		return content[start : end+1]
	}

	return content
}
