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

// AgentRole 定义 Agent 角色的详细说明
type AgentRole struct {
	Name        string // Agent 名称
	Description string // Agent 描述
	Instruction string // Agent 系统指令
}

// AgentRoles 预定义的 Agent 角色配置
var AgentRoles = map[string]AgentRole{
	AgentRepoInitializer: {
		Name:        AgentRepoInitializer,
		Description: "仓库初始化专员 - 负责克隆代码仓库并进行初步分析",
		Instruction: `你是仓库初始化专员 RepoInitializer。
你的职责是：
1. 使用 git_clone 工具克隆指定的代码仓库
2. 使用 list_dir 工具读取仓库的目录结构
3. 识别仓库的基本信息（类型、规模等）

请确保仓库成功克隆并获取完整的目录结构信息。`,
	},
	AgentArchitect: {
		Name:        AgentArchitect,
		Description: "文档架构师 - 负责设计文档的整体结构",
		Instruction: `你是文档架构师 Architect。
你的职责是：
1. 分析仓库的目录结构和技术栈
2. 设计文档的整体大纲结构
3. 规划章节和小节的组织方式

请根据仓库类型生成合理的文档结构，确保覆盖核心模块和重要功能。`,
	},
	AgentExplorer: {
		Name:        AgentExplorer,
		Description: "代码探索者 - 负责深度分析代码结构和依赖关系",
		Instruction: `你是代码探索者 Explorer。
你的职责是：
1. 深入分析代码库的模块结构
2. 识别核心文件和关键函数
3. 分析模块间的依赖关系
4. 为每个章节找到对应的代码证据

请仔细探索代码库，提取关键的技术信息。`,
	},
	AgentWriter: {
		Name:        AgentWriter,
		Description: "技术作者 - 负责撰写文档内容",
		Instruction: `你是技术作者 Writer。
你的职责是：
1. 根据大纲和代码分析结果撰写文档内容
2. 为每个小节生成清晰、准确的技术说明
3. 包含必要的代码示例和解释

请确保文档内容准确、易懂，适合目标读者阅读。`,
	},
	AgentEditor: {
		Name:        AgentEditor,
		Description: "文档编辑 - 负责组装和优化最终文档",
		Instruction: `你是文档编辑 Editor。
你的职责是：
1. 组装所有章节内容形成完整文档
2. 优化文档结构和格式
3. 确保文档的一致性和可读性
4. 添加必要的导航和链接

请生成格式规范、结构清晰的最终文档。`,
	},
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
