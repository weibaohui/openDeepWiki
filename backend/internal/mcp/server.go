package mcp

import (
	"context"
	"encoding/json"
	"fmt"

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
		server.WithToolCapabilities(false),
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

// registerTools 注册所有 MCP 工具
func (w *MCPServerWrapper) registerTools() {
	// 1. list_repositories - 列出所有可用的代码仓库
	w.server.AddTool(mcp.NewTool("list_repositories",
		mcp.WithDescription("列出所有可用的代码仓库。返回仓库列表，包含名称、URL、状态、文档数量等信息。"),
	), w.handleListRepositories)

	// 2. get_repository - 获取仓库详情，包含文档列表
	w.server.AddTool(mcp.NewTool("get_repository",
		mcp.WithDescription("获取仓库详情，包含该仓库下的所有文档列表。可以通过 repo_id 或 repo_name 查询。"),
		mcp.WithNumber("repo_id",
			mcp.Description("仓库ID（数字）"),
		),
		mcp.WithString("repo_name",
			mcp.Description("仓库名称（字符串，与 repo_id 二选一）"),
		),
	), w.handleGetRepository)

	// 3. search_documents - 搜索文档内容
	w.server.AddTool(mcp.NewTool("search_documents",
		mcp.WithDescription("搜索文档内容。通过关键词在文档标题、文件名、内容中进行匹配搜索。可以限定在特定仓库内搜索，也可以全局搜索。"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("搜索关键词"),
		),
		mcp.WithNumber("repo_id",
			mcp.Description("限定仓库ID（可选，不传则搜索所有仓库）"),
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

	klog.V(6).Info("MCP 工具注册完成")
}

// handleListRepositories 处理 list_repositories 工具调用
func (w *MCPServerWrapper) handleListRepositories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	klog.V(6).Info("MCP: handleListRepositories called")

	repos, err := w.repoService.List()
	if err != nil {
		klog.Errorf("MCP: 获取仓库列表失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("获取仓库列表失败: %v", err)), nil
	}

	// 构建返回结果
	var repoInfos []map[string]interface{}
	for _, repo := range repos {
		// 获取每个仓库的文档数量
		docCount := 0
		docs, err := w.docService.GetByRepository(repo.ID)
		if err != nil {
			klog.V(6).Infof("MCP: 获取仓库 %d 文档列表失败: %v", repo.ID, err)
		} else {
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

	var repo *model.Repository
	var err error

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
		docInfos = append(docInfos, map[string]interface{}{
			"id":         doc.ID,
			"title":      doc.Title,
			"filename":   doc.Filename,
			"sort_order": doc.SortOrder,
		})
	}

	result := map[string]interface{}{
		"id":          repo.ID,
		"name":        repo.Name,
		"url":         repo.URL,
		"branch":      repo.CloneBranch,
		"commit":      repo.CloneCommit,
		"status":      repo.Status,
		"documents":   docInfos,
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

	results, err := w.docService.SearchDocuments(ctx, query, repoIDPtr)
	if err != nil {
		klog.Errorf("MCP: 搜索文档失败: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("搜索文档失败: %v", err)), nil
	}

	result := map[string]interface{}{
		"results": results,
		"total":   len(results),
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
