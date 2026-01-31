 openDeepWiki Repo 解读 Workflow（基于 CloudWeGo Eino 的示意实现）
 目标：展示 Agent / Skill / Tool / 调度 的工程化落地方式
 说明：这是一个“结构正确、可实现导向”的示例，而非可直接运行的完整代码

```go

package repodoc

import (
    "context"

    "github.com/cloudwego/eino/agent"
    "github.com/cloudwego/eino/components/llm"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/workflow"
)

// -----------------------------
// 一、全局上下文（Workflow State）
// -----------------------------

// RepoDocState 是整个仓库解读流程共享的“记忆体”
// - 短期记忆：当前章节、小节
// - 中期记忆：仓库分析结果、目录结构
// - 长期记忆：最终 Wiki 文档（可持久化）

type RepoDocState struct {
    RepoURL      string
    LocalPath    string

    RepoType     string   // go / java / frontend / mixed
    TechStack    []string // gin / spring / react ...

    Outline      []ChapterOutline

    CurrentChapter ChapterOutline
    CurrentSection SectionOutline

    DraftContent map[string]string // key = chapter/section
}

// -----------------------------
// 二、结构定义（目录 / 章节 / 小节）
// -----------------------------

type ChapterOutline struct {
    Title    string
    Sections []SectionOutline
}

type SectionOutline struct {
    Title string
    Hints []string // 写作提示 / 关注点
}

// -----------------------------
// 三、Tools（最底层：真正干活）
// -----------------------------

// GitCloneTool：clone 仓库
var GitCloneTool = tool.NewTool(
    "git_clone",
    func(ctx context.Context, repoURL string) (string, error) {
        // exec: git clone xxx
        // return local path
        return "/tmp/repo", nil
    },
)

// ReadRepoTreeTool：读取目录结构
var ReadRepoTreeTool = tool.NewTool(
    "read_repo_tree",
    func(ctx context.Context, path string) (string, error) {
        // exec: tree -L 4
        return "repo tree text", nil
    },
)

// ReadFileTool：读取文件
var ReadFileTool = tool.NewTool(
    "read_file",
    func(ctx context.Context, filePath string) (string, error) {
        return "file content", nil
    },
)

// -----------------------------
// 四、Skill（原子能力，封装 Tool 或 LLM）
// -----------------------------

// RepoPreReadSkill：仓库预读
func RepoPreReadSkill() workflow.Node {
    return workflow.NewLLMNode(
        "repo_pre_read",
        llm.Prompt(`
你将看到一个代码仓库的目录结构。
请判断：
1. 仓库类型（语言 / 框架）
2. 核心模块
3. 适合的解读方式

输出 JSON：{ repo_type, tech_stack }
`),
    )
}

// OutlineGenerateSkill：生成三级目录（目录 / 章节 / 标题）
func OutlineGenerateSkill() workflow.Node {
    return workflow.NewLLMNode(
        "generate_outline",
        llm.Prompt(`
基于仓库分析结果，生成一份 Wiki 目录：
- Chapter
- Section
- Title

要求：适合技术人员阅读
输出结构化 JSON
`),
    )
}

// SectionExploreSkill：针对某一标题，探索仓库
func SectionExploreSkill() workflow.Node {
    return workflow.NewLLMNode(
        "section_explore",
        llm.Prompt(`
你正在为以下标题写文档：{{.Section.Title}}

请判断需要查看哪些代码 / 文件，并给出理由。
必要时调用工具读取代码。
`),
    )
}

// SectionWriteSkill：正式写作
func SectionWriteSkill() workflow.Node {
    return workflow.NewLLMNode(
        "section_write",
        llm.Prompt(`
请根据已有信息，完成该小节的 Wiki 文档。
要求：
- 技术准确
- 结构清晰
- 面向工程师
`),
    )
}

// GapCheckSkill：差缺补漏
func GapCheckSkill() workflow.Node {
    return workflow.NewLLMNode(
        "gap_check",
        llm.Prompt(`
请检查当前小节内容是否存在：
- 与标题不匹配
- 关键信息缺失
- 与仓库实际不符

如有问题，请指出并给出补充建议。
`),
    )
}

// -----------------------------
// 五、Agent（角色 + 调度）
// -----------------------------

// RepoDocAgent：负责“解读一个代码仓库”
func RepoDocAgent(model llm.ChatModel) *agent.Agent {
    return agent.New(
        agent.WithModel(model),
        agent.WithTools(
            GitCloneTool,
            ReadRepoTreeTool,
            ReadFileTool,
        ),
    )
}

// -----------------------------
// 六、Workflow（调度 Agent / Skill）
// -----------------------------

func BuildRepoDocWorkflow() *workflow.Workflow {
    wf := workflow.New("repo_doc_workflow")

    // Step 1: clone repo
    wf.AddNode("clone", workflow.NewToolNode(GitCloneTool))

    // Step 2: read tree
    wf.AddNode("tree", workflow.NewToolNode(ReadRepoTreeTool))

    // Step 3: repo pre-read
    wf.AddNode("pre_read", RepoPreReadSkill())

    // Step 4: generate outline
    wf.AddNode("outline", OutlineGenerateSkill())

    // Step 5: loop chapters / sections
    wf.AddNode("explore", SectionExploreSkill())
    wf.AddNode("write", SectionWriteSkill())
    wf.AddNode("gap_check", GapCheckSkill())

    // Edges（顺序 + 循环）
    wf.Connect("clone", "tree")
    wf.Connect("tree", "pre_read")
    wf.Connect("pre_read", "outline")

    wf.Connect("outline", "explore")
    wf.Connect("explore", "write")
    wf.Connect("write", "gap_check")
    wf.Connect("gap_check", "explore") // until section done

    return wf
}

// -----------------------------
// 七、你真正得到的能力
// -----------------------------
// 1. 一个“调度 Agent”（Workflow）
// 2. 多个“执行 Agent / Skill”
// 3. 共享 State（记忆）
// 4. Tool 可观测、可替换
// 5. 支持中断 / 恢复 / 并行
```