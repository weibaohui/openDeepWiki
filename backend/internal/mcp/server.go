package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// MCPServerWrapper 封装 mcp-go Server，提供文档查询工具
type MCPServerWrapper struct {
	server      *server.MCPServer
	repoService *service.RepositoryService
	docService  *service.DocumentService
}

// NewMCPServer 创建 MCP Server 实例并注册工具
func NewMCPServer(repoSvc *service.RepositoryService, docSvc *service.DocumentService) *MCPServerWrapper {
	// 创建 mcp-go Server
	s := server.NewMCPServer(
		"openDeepWiki",
		"1.0.0",
		server.WithRecovery(),
	)

	wrapper := &MCPServerWrapper{
		server:      s,
		repoService: repoSvc,
		docService:  docSvc,
	}

	// 注册工具
	wrapper.registerTools()

	klog.V(6).Info("MCP Server 初始化完成")
	return wrapper
}

// GetServer 返回底层的 mcp-go Server
func (w *MCPServerWrapper) GetServer() *server.MCPServer {
	return w.server
}

// logMCPActivity 记录 MCP 活动日志
func logMCPActivity(format string, args ...interface{}) {
	klog.V(6).Infof("[MCP] "+format, args...)
}

// registerTools 注册所有 MCP 工具
func (w *MCPServerWrapper) registerTools() {
	// 1. list_repositories - 列出所有可用的代码仓库
	w.server.AddTool(mcp.NewTool("list_repositories",
		mcp.WithDescription("列出所有可用的代码仓库。返回仓库列表，包含名称、URL、状态、文档数量等信息。支持分页和状态过滤。"),
		mcp.WithNumber("limit",
			mcp.Description("返回结果数量限制，默认 20，最大 100"),
		),
		mcp.WithNumber("offset",
			mcp.Description("分页偏移量，从 0 开始"),
		),
		mcp.WithString("status",
			mcp.Description("按状态过滤，可选值：ready, pending, failed, cloning"),
		),
	), w.handleListRepositories)

	// 2. get_repository - 获取仓库详情，包含文档列表
	w.server.AddTool(mcp.NewTool("get_repository",
		mcp.WithDescription("获取仓库详情，包含该仓库下的所有文档列表。优先使用 repo_id 查询，如不知道 ID 可使用 repo_name。"),
		mcp.WithNumber("repo_id",
			mcp.Description("仓库ID（数字），优先使用此参数"),
		),
		mcp.WithString("repo_name",
			mcp.Description("仓库名称（字符串），仅在不传 repo_id 时使用"),
		),
		mcp.WithBoolean("include_content",
			mcp.Description("是否包含文档完整内容，默认 false（仅返回文档列表）"),
		),
	), w.handleGetRepository)

	// 3. search_documents - 搜索文档内容
	w.server.AddTool(mcp.NewTool("search_documents",
		mcp.WithDescription("搜索文档内容。通过关键词在文档标题、文件名、内容中进行匹配搜索。支持分页和类型过滤。"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索关键词"),
		),
		mcp.WithNumber("repo_id",
			mcp.Description("限定仓库ID（可选，不传则搜索所有仓库）"),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回结果数量限制，默认 10，最大 50"),
		),
		mcp.WithString("doc_type",
			mcp.Description("按文档类型过滤，可选值：api, architecture, guide, readme"),
		),
	), w.handleSearchDocuments)

	// 4. read_document - 读取文档完整内容
	w.server.AddTool(mcp.NewTool("read_document",
		mcp.WithDescription("读取文档的完整内容（Markdown 格式）。返回文档的全部内容，包含元信息如所属仓库、分支、版本等。"),
		mcp.WithNumber("doc_id",
			mcp.Required(),
			mcp.Description("文档ID（数字）"),
		),
	), w.handleReadDocument)

	// 5. get_document_summary - 获取文档摘要
	w.server.AddTool(mcp.NewTool("get_document_summary",
		mcp.WithDescription("获取文档摘要（前 500 字），用于快速判断文档相关性。如需要完整内容，请使用 read_document。"),
		mcp.WithNumber("doc_id",
			mcp.Required(),
			mcp.Description("文档ID（数字）"),
		),
	), w.handleGetDocumentSummary)

	klog.V(6).Info("MCP 工具注册完成")
}

// handleListRepositories 处理 list_repositories 工具调用
func (w *MCPServerWrapper) handleListRepositories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Info("MCP: handleListRepositories called")

	// 解析分页参数
	limit := 20
	if l, err := request.RequireInt("limit"); err == nil && l > 0 {
		limit = l
		if limit > 100 {
			limit = 100
		}
	}

	offset := 0
	if o, err := request.RequireInt("offset"); err == nil && o >= 0 {
		offset = o
	}

	status := request.GetString("status", "")

	repos, err := w.repoService.List()
	if err != nil {
		klog.Errorf("MCP: 获取仓库列表失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("获取仓库列表失败: %v", err)), nil
	}

	// 状态过滤
	if status != "" {
		var filtered []model.Repository
		for _, r := range repos {
			if r.Status == status {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
	}

	total := len(repos)

	// 分页
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	// 构建返回结果
	var repoInfos []map[string]interface{}
	for _, repo := range repos[start:end] {
		docCount := 0
		if docs, err := w.docService.GetByRepository(repo.ID); err == nil {
			docCount = len(docs)
		}
		repoInfos = append(repoInfos, map[string]interface{}{
			"id":             repo.ID,
			"name":           repo.Name,
			"url":            repo.URL,
			"status":         repo.Status,
			"branch":         repo.CloneBranch,
			"document_count": docCount,
			"updated_at":     repo.UpdatedAt,
		})
	}

	result := map[string]interface{}{
		"repositories": repoInfos,
		"total":        total,
		"has_more":     end < total,
		"limit":        limit,
		"offset":       offset,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetRepository 处理 get_repository 工具调用
func (w *MCPServerWrapper) handleGetRepository(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Infof("MCP: handleGetRepository called, params: %+v", request.Params.Arguments)

	// 解析参数
	repoID, _ := request.RequireInt("repo_id")
	repoName := request.GetString("repo_name", "")
	includeContent := request.GetBool("include_content", false)

	var repo *model.Repository
	var err error

	// 优先使用 repo_id
	if repoID > 0 {
		repo, err = w.repoService.Get(uint(repoID))
		if err != nil {
			klog.Errorf("MCP: 获取仓库失败 (id=%d): %v", repoID, err)
			return mcp.NewToolResultError(fmt.Sprintf("获取仓库失败: %v", err)), nil
		}
		if repo == nil {
			return mcp.NewToolResultError(fmt.Sprintf("未找到ID为 %d 的仓库", repoID)), nil
		}
	} else if repoName != "" {
		// 通过名称查找仓库
		repos, listErr := w.repoService.List()
		if listErr != nil {
			klog.Errorf("MCP: 获取仓库列表失败: %v", listErr)
			return mcp.NewToolResultError(fmt.Sprintf("获取仓库列表失败: %v", listErr)), nil
		}
		for _, r := range repos {
			if r.Name == repoName {
				repo = &r
				break
			}
		}
		if repo == nil {
			return mcp.NewToolResultError(fmt.Sprintf("未找到名称为 '%s' 的仓库", repoName)), nil
		}
	} else {
		return mcp.NewToolResultError("需要提供 repo_id 或 repo_name 参数"), nil
	}

	// 获取文档列表
	docs, err := w.docService.GetByRepository(repo.ID)
	if err != nil {
		klog.Errorf("MCP: 获取文档列表失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("获取文档列表失败: %v", err)), nil
	}

	// 构建简化的文档信息
	var docInfos []map[string]interface{}
	for _, doc := range docs {
		docInfo := map[string]interface{}{
			"id":       doc.ID,
			"title":    doc.Title,
			"filename": doc.Filename,
		}
		if includeContent {
			docInfo["content"] = doc.Content
		}
		docInfos = append(docInfos, docInfo)
	}

	result := map[string]interface{}{
		"id":        repo.ID,
		"name":      repo.Name,
		"url":       repo.URL,
		"branch":    repo.CloneBranch,
		"commit":    repo.CloneCommit,
		"status":    repo.Status,
		"documents": docInfos,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// handleSearchDocuments 处理 search_documents 工具调用
func (w *MCPServerWrapper) handleSearchDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Infof("MCP: handleSearchDocuments called, params: %+v", request.Params.Arguments)

	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query 参数是必需的"), nil
	}

	repoID, _ := request.RequireInt("repo_id")
	var repoIDPtr *uint
	if repoID > 0 {
		id := uint(repoID)
		repoIDPtr = &id
	}

	// 分页参数
	limit := 10
	if l, err := request.RequireInt("limit"); err == nil && l > 0 {
		limit = l
		if limit > 50 {
			limit = 50
		}
	}

	docType := request.GetString("doc_type", "")

	results, err := w.docService.SearchDocuments(ctx, query, repoIDPtr)
	if err != nil {
		klog.Errorf("MCP: 搜索文档失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("搜索文档失败: %v", err)), nil
	}

	// 按文档类型过滤
	if docType != "" {
		var filtered []service.DocumentSearchResult
		for _, r := range results {
			if strings.Contains(strings.ToLower(r.Filename), strings.ToLower(docType)) {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	total := len(results)

	// 限制返回数量
	if limit > total {
		limit = total
	}

	result := map[string]interface{}{
		"results":  results[:limit],
		"total":    total,
		"has_more": limit < total,
		"limit":    limit,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// handleReadDocument 处理 read_document 工具调用
func (w *MCPServerWrapper) handleReadDocument(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Infof("MCP: handleReadDocument called, params: %+v", request.Params.Arguments)

	docID, err := request.RequireInt("doc_id")
	if err != nil {
		return mcp.NewToolResultError("doc_id 参数是必需的"), nil
	}

	doc, err := w.docService.Get(uint(docID))
	if err != nil {
		klog.Errorf("MCP: 获取文档失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("获取文档失败: %v", err)), nil
	}

	// 获取仓库名称
	repoName := ""
	repo, err := w.repoService.Get(doc.RepositoryID)
	if err == nil && repo != nil {
		repoName = repo.Name
	}

	result := map[string]interface{}{
		"id":        doc.ID,
		"repo_id":   doc.RepositoryID,
		"repo_name": repoName,
		"title":     doc.Title,
		"filename":  doc.Filename,
		"content":   doc.Content,
		"branch":    doc.CloneBranch,
		"commit":    doc.CloneCommitID,
		"version":   doc.Version,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetDocumentSummary 处理 get_document_summary 工具调用
func (w *MCPServerWrapper) handleGetDocumentSummary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Infof("MCP: handleGetDocumentSummary called, params: %+v", request.Params.Arguments)

	docID, err := request.RequireInt("doc_id")
	if err != nil {
		return mcp.NewToolResultError("doc_id 参数是必需的"), nil
	}

	doc, err := w.docService.Get(uint(docID))
	if err != nil {
		klog.Errorf("MCP: 获取文档失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("获取文档失败: %v", err)), nil
	}

	// 获取仓库名称
	repoName := ""
	repo, err := w.repoService.Get(doc.RepositoryID)
	if err == nil && repo != nil {
		repoName = repo.Name
	}

	// 生成摘要（前 500 字符）
	summary := doc.Content
	if len(summary) > 500 {
		summary = summary[:500] + "..."
	}

	result := map[string]interface{}{
		"id":               doc.ID,
		"title":            doc.Title,
		"filename":         doc.Filename,
		"repo_id":          doc.RepositoryID,
		"repo_name":        repoName,
		"summary":          summary,
		"total_length":     len(doc.Content),
		"has_full_content": len(doc.Content) > 500,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
