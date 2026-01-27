package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opendeepwiki/backend/pkg/llm"
)

type LLMAnalyzer struct {
	client *llm.Client
}

func NewLLMAnalyzer(client *llm.Client) *LLMAnalyzer {
	return &LLMAnalyzer{client: client}
}

type AnalyzeRequest struct {
	TaskType    string       `json:"task_type"`
	ProjectInfo *ProjectInfo `json:"project_info"`
}

func (a *LLMAnalyzer) Analyze(ctx context.Context, req AnalyzeRequest) (string, error) {
	systemPrompt := getSystemPrompt(req.TaskType)
	userPrompt := buildUserPrompt(req)

	return a.client.GenerateDocument(ctx, systemPrompt, userPrompt)
}

func getSystemPrompt(taskType string) string {
	basePrompt := `你是一个专业的代码分析师，擅长分析和解读各种编程语言的代码仓库。
你需要根据提供的项目信息，生成清晰、专业的 Markdown 格式文档。
文档应该结构清晰，内容准确，易于理解。
请使用中文撰写文档。`

	prompts := map[string]string{
		"overview": basePrompt + `

你的任务是生成【项目概览】文档，需要包含：
1. 项目简介 - 项目的主要功能和目标
2. 技术栈 - 使用的编程语言、框架和主要依赖
3. 目录结构 - 项目的目录组织方式及各目录的用途
4. 快速开始 - 如何安装和运行项目（如果能推断出来）

文档格式要求：
- 使用 Markdown 格式
- 包含清晰的标题层级
- 适当使用代码块、表格、列表等元素`,

		"architecture": basePrompt + `

你的任务是生成【架构分析】文档，需要包含：
1. 整体架构 - 项目的整体架构设计
2. 模块划分 - 各模块的职责和功能
3. 依赖关系 - 模块之间的依赖和调用关系
4. 设计模式 - 项目中使用的设计模式（如果能识别出）
5. 数据流 - 数据在系统中的流转方式

请根据目录结构和关键文件推断项目架构。`,

		"api": basePrompt + `

你的任务是生成【核心接口】文档，需要包含：
1. API 概览 - 主要的 API 接口或公开函数
2. 接口详情 - 重要接口的参数、返回值说明
3. 调用示例 - 典型的使用方式
4. 模块导出 - 各模块对外暴露的接口

请根据入口文件和关键代码文件分析接口设计。`,

		"business-flow": basePrompt + `

你的任务是生成【业务流程】文档，需要包含：
1. 核心业务 - 项目要解决的主要业务问题
2. 业务流程 - 主要的业务处理流程
3. 数据模型 - 核心的数据结构和模型
4. 关键算法 - 重要的算法或处理逻辑（如果有）

请根据项目结构和关键文件推断业务逻辑。`,

		"deployment": basePrompt + `

你的任务是生成【部署配置】文档，需要包含：
1. 环境要求 - 运行项目所需的环境和依赖
2. 配置说明 - 配置文件和环境变量说明
3. 部署方式 - 如何部署项目（Docker、K8s、直接运行等）
4. 运维建议 - 日志、监控、备份等建议

请根据配置文件和部署相关文件生成文档。`,
	}

	if prompt, ok := prompts[taskType]; ok {
		return prompt
	}
	return basePrompt
}

func buildUserPrompt(req AnalyzeRequest) string {
	info := req.ProjectInfo
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 项目信息\n\n"))
	sb.WriteString(fmt.Sprintf("**项目名称**: %s\n", info.Name))
	sb.WriteString(fmt.Sprintf("**项目类型**: %s\n", info.Type))
	sb.WriteString(fmt.Sprintf("**文件总数**: %d\n", info.TotalFiles))
	sb.WriteString(fmt.Sprintf("**代码行数**: %d\n\n", info.TotalLines))

	sb.WriteString("## 语言分布\n\n")
	for lang, count := range info.Languages {
		sb.WriteString(fmt.Sprintf("- %s: %d 个文件\n", lang, count))
	}
	sb.WriteString("\n")

	sb.WriteString("## 目录结构\n\n```\n")
	sb.WriteString(formatDirectoryTree(info.Structure, "", true))
	sb.WriteString("```\n\n")

	if len(info.KeyFiles) > 0 {
		sb.WriteString("## 关键文件\n\n")
		for _, kf := range info.KeyFiles {
			sb.WriteString(fmt.Sprintf("### %s (%s)\n", kf.Path, kf.Description))
			if kf.Preview != "" {
				sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", kf.Preview))
			}
		}
	}

	if len(info.Dependencies) > 0 {
		sb.WriteString("## 主要依赖\n\n")
		for _, dep := range info.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s\n", dep))
		}
		sb.WriteString("\n")
	}

	if len(info.EntryPoints) > 0 {
		sb.WriteString("## 入口文件\n\n")
		for _, entry := range info.EntryPoints {
			sb.WriteString(fmt.Sprintf("- %s\n", entry))
		}
		sb.WriteString("\n")
	}

	if len(info.ConfigFiles) > 0 {
		sb.WriteString("## 配置文件\n\n")
		for _, cfg := range info.ConfigFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", cfg))
		}
		sb.WriteString("\n")
	}

	if info.ReadmeContent != "" {
		sb.WriteString("## README 内容\n\n")
		sb.WriteString(info.ReadmeContent)
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n\n请根据以上信息生成文档。")

	return sb.String()
}

func formatDirectoryTree(tree *DirectoryTree, prefix string, isLast bool) string {
	if tree == nil {
		return ""
	}

	var sb strings.Builder

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	if tree.Path != "" {
		sb.WriteString(prefix + connector + tree.Name)
		if tree.IsDir {
			sb.WriteString("/")
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(tree.Name + "/\n")
	}

	if tree.Children != nil {
		newPrefix := prefix
		if tree.Path != "" {
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
		}

		for i, child := range tree.Children {
			isLastChild := i == len(tree.Children)-1
			sb.WriteString(formatDirectoryTree(child, newPrefix, isLastChild))
		}
	}

	return sb.String()
}

func ProjectInfoToJSON(info *ProjectInfo) string {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
