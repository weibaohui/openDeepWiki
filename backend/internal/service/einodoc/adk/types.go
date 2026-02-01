// Package adk 基于 Eino ADK 的 SequentialAgent 模式实现
// 提供多 Agent 协作的代码仓库文档生成能力
package adk

import (
	"github.com/opendeepwiki/backend/internal/service/einodoc"
)

// WorkflowInput Workflow 输入
// 作为 SequentialAgent 的输入类型
type WorkflowInput struct {
	RepoURL string `json:"repo_url"` // 仓库 Git URL
}

// WorkflowOutput Workflow 输出
// 作为 SequentialAgent 的输出类型
type WorkflowOutput struct {
	State  *einodoc.RepoDocState  `json:"state"`  // Workflow 执行过程中的状态
	Result *einodoc.RepoDocResult `json:"result"` // 最终结果
}

// RepoAnalysis 仓库分析结果
type RepoAnalysis struct {
	RepoType  string   `json:"repo_type"`  // 仓库类型: go / java / python / frontend / mixed
	TechStack []string `json:"tech_stack"` // 技术栈列表
	Summary   string   `json:"summary"`    // 仓库简介
}

// DocOutline 文档大纲
type DocOutline struct {
	Chapters []einodoc.Chapter `json:"chapters"` // 章节列表
}

// SectionContent 小节内容
type SectionContent struct {
	ChapterIdx int    `json:"chapter_idx"` // 章节索引
	SectionIdx int    `json:"section_idx"` // 小节索引
	Content    string `json:"content"`     // 内容
}

// WorkflowInfo Workflow 信息
type WorkflowInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Agents      []string `json:"agents"`
}
