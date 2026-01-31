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
)

// WorkflowInput Workflow 输入
type WorkflowInput struct {
	RepoURL string `json:"repo_url"`
}

// WorkflowOutput Workflow 输出
type WorkflowOutput struct {
	State  *RepoDocState  `json:"state"`
	Result *RepoDocResult `json:"result"`
}

// RepoDocChain 基于 Chain 的简化 Workflow
type RepoDocChain struct {
	chain *compose.Chain[WorkflowInput, WorkflowOutput]
}

// NewRepoDocChain 创建 RepoDoc Chain
func NewRepoDocChain(basePath string, chatModel model.ChatModel) (*RepoDocChain, error) {
	chain := compose.NewChain[WorkflowInput, WorkflowOutput]()

	// Step 1: Clone & Read Tree
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input WorkflowInput) (WorkflowOutput, error) {
		state := NewRepoDocState(input.RepoURL, "")

		// 使用 git_clone 工具克隆仓库
		cloneTool := NewGitCloneTool(basePath)
		cloneArgs, _ := json.Marshal(map[string]string{
			"repo_url":   input.RepoURL,
			"target_dir": generateRepoDirName(input.RepoURL),
		})
		cloneResult, err := cloneTool.InvokableRun(ctx, string(cloneArgs))
		if err != nil {
			return WorkflowOutput{}, fmt.Errorf("clone failed: %w", err)
		}

		state.LocalPath = filepath.Join(basePath, generateRepoDirName(input.RepoURL))
		_ = cloneResult

		// 读取目录结构
		listTool := NewListDirTool(basePath)
		listArgs, _ := json.Marshal(map[string]interface{}{
			"dir":       generateRepoDirName(input.RepoURL),
			"recursive": true,
		})
		treeResult, err := listTool.InvokableRun(ctx, string(listArgs))
		if err != nil {
			return WorkflowOutput{}, fmt.Errorf("list dir failed: %w", err)
		}

		// 将 treeResult 存储到 state 中供后续使用
		_ = treeResult

		return WorkflowOutput{
			State: state,
			Result: &RepoDocResult{
				RepoURL:   input.RepoURL,
				LocalPath: state.LocalPath,
				Outline:   []Chapter{},
			},
		}, nil
	}))

	// Step 2: Pre-read Analysis with LLM
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		state := output.State

		// 读取目录结构
		listTool := NewListDirTool(basePath)
		listArgs, _ := json.Marshal(map[string]interface{}{
			"dir":       generateRepoDirName(state.RepoURL),
			"recursive": true,
		})
		treeResult, err := listTool.InvokableRun(ctx, string(listArgs))
		if err != nil {
			treeResult = "Failed to read directory"
		}

		// 使用 LLM 分析仓库
		messages := []*schema.Message{
			{
				Role:    schema.System,
				Content: "You are a code repository analyzer. Analyze the repository structure and provide: 1) Repository type (go/java/python/frontend/mixed), 2) Tech stack, 3) Brief summary. Respond in JSON format with fields: repo_type, tech_stack (array), summary",
			},
			{
				Role:    schema.User,
				Content: fmt.Sprintf("Repository URL: %s\n\nDirectory Structure:\n%s", state.RepoURL, treeResult),
			},
		}

		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			// 使用默认值
			state.SetRepoInfo("unknown", []string{})
		} else {
			// 解析 JSON 响应
			var analysis struct {
				RepoType  string   `json:"repo_type"`
				TechStack []string `json:"tech_stack"`
				Summary   string   `json:"summary"`
			}
			content := extractJSON(resp.Content)
			if err := json.Unmarshal([]byte(content), &analysis); err != nil {
				state.SetRepoInfo("unknown", []string{})
			} else {
				state.SetRepoInfo(analysis.RepoType, analysis.TechStack)
			}
		}

		output.Result.RepoType = state.RepoType
		output.Result.TechStack = state.TechStack

		return output, nil
	}))

	// Step 3: Generate Outline with LLM
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		state := output.State

		messages := []*schema.Message{
			{
				Role: schema.System,
				Content: `You are a technical documentation expert. Create a documentation outline for the repository.

Respond in JSON format:
{
  "chapters": [
    {
      "title": "Chapter Title",
      "sections": [
        {"title": "Section Title", "hints": ["hint1", "hint2"]}
      ]
    }
  ]
}`,
			},
			{
				Role: schema.User,
				Content: fmt.Sprintf("Repository Type: %s\nTech Stack: %v\n\nGenerate a documentation outline with 2-3 chapters, each with 2-3 sections.",
					state.RepoType, state.TechStack),
			},
		}

		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			// 使用默认大纲
			state.SetOutline([]Chapter{
				{
					Title: "Overview",
					Sections: []Section{
						{Title: "Introduction", Hints: []string{"Project overview"}},
						{Title: "Architecture", Hints: []string{"System architecture"}},
					},
				},
			})
		} else {
			var outline struct {
				Chapters []Chapter `json:"chapters"`
			}
			content := extractJSON(resp.Content)
			if err := json.Unmarshal([]byte(content), &outline); err != nil {
				state.SetOutline([]Chapter{
					{
						Title: "Overview",
						Sections: []Section{
							{Title: "Introduction", Hints: []string{"Project overview"}},
						},
					},
				})
			} else {
				state.SetOutline(outline.Chapters)
			}
		}

		output.Result.Outline = state.Outline
		return output, nil
	}))

	// Step 4: Generate Section Content (Simplified)
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		state := output.State

		// 为每个 section 生成简单的内容
		for chIdx, chapter := range state.Outline {
			for secIdx, section := range chapter.Sections {
				_ = sectionKey(chIdx, secIdx) // 避免未使用变量

				// 使用 LLM 生成内容
				messages := []*schema.Message{
					{
						Role:    schema.System,
						Content: "You are a technical writer. Write a brief documentation section in Markdown format.",
					},
					{
						Role: schema.User,
						Content: fmt.Sprintf("Chapter: %s\nSection: %s\nHints: %v\n\nWrite a short paragraph about this topic.",
							chapter.Title, section.Title, section.Hints),
					},
				}

				resp, err := chatModel.Generate(ctx, messages)
				if err != nil {
					state.SetSectionContent(chIdx, secIdx, fmt.Sprintf("## %s\n\nContent for %s under %s.\n\n*Generated by Eino Workflow*",
						section.Title, section.Title, chapter.Title))
				} else {
					state.SetSectionContent(chIdx, secIdx, resp.Content)
				}
			}
		}

		return output, nil
	}))

	// Step 5: Finalize Document
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, output WorkflowOutput) (WorkflowOutput, error) {
		state := output.State

		// 组装最终文档
		var doc strings.Builder
		doc.WriteString(fmt.Sprintf("# Project Documentation\n\n"))
		doc.WriteString(fmt.Sprintf("**Repository:** %s\n\n", state.RepoURL))
		doc.WriteString(fmt.Sprintf("**Type:** %s\n\n", state.RepoType))
		doc.WriteString(fmt.Sprintf("**Tech Stack:** %v\n\n", state.TechStack))
		doc.WriteString("---\n\n")

		for chIdx, chapter := range state.Outline {
			doc.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))

			for secIdx, section := range chapter.Sections {
				doc.WriteString(fmt.Sprintf("### %s\n\n", section.Title))

				content := state.GetSectionContent(chIdx, secIdx)
				if content != "" {
					doc.WriteString(content)
					doc.WriteString("\n\n")
				}
			}
		}

		state.SetFinalDocument(doc.String())
		output.Result.Document = doc.String()
		output.Result.SectionsContent = state.SectionsContent
		output.Result.SectionsCount = len(state.SectionsContent)
		output.Result.Completed = true

		return output, nil
	}))

	return &RepoDocChain{chain: chain}, nil
}

// Run 执行 Chain
func (c *RepoDocChain) Run(ctx context.Context, input WorkflowInput) (*RepoDocResult, error) {
	runnable, err := c.chain.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile chain: %w", err)
	}

	output, err := runnable.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("chain execution failed: %w", err)
	}

	return output.Result, nil
}

// extractJSON 从文本中提取 JSON 部分
func extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start >= 0 && end > start {
		return content[start : end+1]
	}

	return content
}
