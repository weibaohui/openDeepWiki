# Eino RepoDoc Service 设计文档

## 背景

基于决策文档 `003-Eino-框架-调研.md`、`004-Eino-Workflow-思路.md` 和 `004-Eino-Workflow-示例.md`，本设计实现一个基于 CloudWeGo Eino 框架的仓库文档解析服务示例。

## 目标

1. 封装为 Service，输入 repo 地址，输出解析文档
2. 使用 Eino Workflow 编排流程
3. 复用项目已有的 Tools 和 Agent 实现
4. 流程能跑起来即可，不追求内容完全准确

## 架构设计

### 核心组件映射

| 当前概念 | Eino 对应组件 | 职责 |
|---------|--------------|------|
| Tool | Tool Node | 执行具体操作（clone、read file 等） |
| Skill | LLM Node / Lambda Node | 原子认知能力（分析、生成） |
| Agent | ChatModelAgent | 拥有工具 + LLM 的执行体 |
| 调度 | Workflow | 指挥流程，不写内容 |

### Workflow 流程设计

```
RepoDocWorkflow
├── Step 1: Clone Repo (Tool Node)
│   └── 使用 git_clone 工具克隆仓库
├── Step 2: Read Tree (Tool Node)
│   └── 使用 list_dir 工具读取目录结构
├── Step 3: Pre-read Analysis (LLM Node)
│   └── 分析仓库类型、技术栈
├── Step 4: Generate Outline (LLM Node)
│   └── 生成文档大纲
├── Step 5: Explore & Write (Loop Branch)
│   ├── Section Explore (LLM Node)
│   │   └── 探索需要查看的文件
│   ├── Read Files (Tool Node)
│   │   └── 读取相关文件
│   └── Section Write (LLM Node)
│       └── 生成小节内容
└── Step 6: Finalize (Lambda Node)
    └── 汇总所有内容生成最终文档
```

### State 设计

```go
type RepoDocState struct {
    // 输入
    RepoURL   string
    LocalPath string
    
    // 分析结果
    RepoType  string   // 仓库类型
    TechStack []string // 技术栈
    
    // 大纲
    Outline []Chapter
    
    // 当前处理状态
    CurrentChapterIdx int
    CurrentSectionIdx int
    
    // 输出
    SectionsContent map[string]string // 章节内容
    FinalDocument   string            // 最终文档
}

type Chapter struct {
    Title    string
    Sections []Section
}

type Section struct {
    Title string
    Hints []string
}
```

## 实现细节

### 1. Service 接口

```go
type RepoDocService interface {
    ParseRepo(ctx context.Context, repoURL string) (*RepoDocResult, error)
}
```

### 2. 工具复用

复用已有的 tools 实现：
- `git_clone` - 克隆仓库
- `list_dir` - 列出目录
- `read_file` - 读取文件
- `search_files` - 搜索文件

### 3. LLM 交互

复用已有的 LLM client：
- `ChatWithToolExecution` - 带工具调用的对话

## 文件结构

```
backend/internal/service/einodoc/
├── service.go      # Service 接口和实现
├── workflow.go     # Eino Workflow 定义
├── state.go        # State 类型定义
└── nodes.go        # 各个 Node 的实现
```

## 测试计划

1. 编译通过
2. 能运行 workflow（使用 mock 数据或真实 LLM）
3. 输出结构化的文档（不要求内容完全准确）
