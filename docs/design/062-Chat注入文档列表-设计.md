# Chat 注入文档列表设计

## 变更记录表

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2025-03-04 | 初始设计文档创建 | AI Assistant |

---

## 1. 需求背景

当前 Chat 功能在建立连接时，只注入了仓库的基本信息（名称、地址、分支等）。为了让智能体更好地理解仓库内容，需要在会话开始时注入仓库下的文档列表（标题和ID），使智能体能够：
1. 先查阅文档了解仓库结构
2. 根据用户问题匹配相关文档
3. 必要时通过 `read_doc(doc_id)` 获取全文

## 2. 设计目标

在 `chat_handler.go` 的 `runAgent` 方法中，追加当前 Repo 下的文档标题和 ID 列表到 `repoInfo` 变量。

## 3. 详细设计

### 3.1 代码变更范围

**修改文件:**
1. `backend/internal/handler/chat_handler.go` - 添加 docService 依赖，修改 repoInfo 构建
2. `backend/cmd/server/main.go` - 传入 docService 到 NewChatHandler

### 3.2 ChatHandler 结构变更

```go
// Before
type ChatHandler struct {
    chatService  service.ChatService
    repoService  *service.RepositoryService
    hub          *ChatHub
    agentFactory *adkagents.AgentFactory
}

// After - 添加 docService
type ChatHandler struct {
    chatService  service.ChatService
    repoService  *service.RepositoryService
    docService   *service.DocumentService  // 新增
    hub          *ChatHub
    agentFactory *adkagents.AgentFactory
}
```

### 3.3 NewChatHandler 签名变更

```go
// Before
func NewChatHandler(
    chatService service.ChatService,
    repoService *service.RepositoryService,
    agentFactory *adkagents.AgentFactory,
) *ChatHandler

// After - 添加 docService 参数
func NewChatHandler(
    chatService service.ChatService,
    repoService *service.RepositoryService,
    docService *service.DocumentService,  // 新增
    agentFactory *adkagents.AgentFactory,
) *ChatHandler
```

### 3.4 repoInfo 构建逻辑变更

在 `runAgent` 方法中，获取仓库信息后，追加文档列表：

```go
// 获取仓库信息
var repoInfo string
if h.repoService != nil {
    repo, err := h.repoService.Get(client.repoID)
    if err == nil && repo != nil {
        repoInfo = fmt.Sprintf("## 当前仓库信息\n- 仓库名称: %s\n- 仓库地址: %s\n- 本地路径: %s\n- 仓库描述: %s\n- 当前分支: %s\n- 当前Commit: %s\n",
            repo.Name, repo.URL, repo.LocalPath, repo.Description, repo.CloneBranch, repo.CloneCommit)

        // 新增：追加文档列表
        if h.docService != nil {
            docs, err := h.docService.GetByRepository(client.repoID)
            if err == nil && len(docs) > 0 {
                repoInfo += "\n## 文档列表\n"
                for _, doc := range docs {
                    repoInfo += fmt.Sprintf("- 标题: %s, DocID: %d\n", doc.Title, doc.ID)
                }
                repoInfo += "\n可根据原始文档DocID，通过read_doc(doc_id)获取原文全文\n"
            }
        }
    }
}
```

### 3.5 main.go 变更

```go
// Before (第217行)
chatHandler := handler.NewChatHandler(chatService, repoService, agentFactory)

// After - 传入 docService
chatHandler := handler.NewChatHandler(chatService, repoService, docService, agentFactory)
```

## 4. 影响分析

- **依赖变更**: ChatHandler 新增对 DocumentService 的依赖
- **API 变更**: NewChatHandler 函数签名变更（需要同步修改调用方）
- **功能增强**: 智能体现在可以了解仓库下的所有文档，优先通过文档而非源码回答用户问题
- **向后兼容**: 如果 docService 为 nil，功能回退到原有行为

## 5. 测试要点

1. 编译通过
2. 建立 Chat 连接时，repoInfo 正确包含文档列表
3. 空仓库（无文档）时正常处理
4. docService 为 nil 时正常处理（防御性编程）

## 6. 与 Agent 定义的对应关系

Agent 定义中已添加：
```
- 文档信息:
  1. 文档标题、DocID
  2. 根据原始文档DocID，可通过read_doc(doc_id)获取的原文全文
```

本实现为 Agent 提供了文档标题和 DocID 列表，使 Agent 可以：
1. 理解仓库文档结构
2. 根据用户问题匹配相关文档
3. 调用 `read_doc(doc_id)` 获取需要的文档全文
