package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	vectorservice "github.com/weibaohui/opendeepwiki/backend/internal/service/vector"
)

// VectorHandler 向量搜索处理器
type VectorHandler struct {
	searchService    *vectorservice.VectorSearchService
	embeddingService *vectorservice.VectorEmbeddingService
	vectorRepo       repository.VectorRepository
	vectorTaskRepo   repository.VectorTaskRepository
	repoRepo         repository.RepoRepository
	docRepo          repository.DocumentRepository
}

// NewVectorHandler 创建向量处理器
func NewVectorHandler(
	searchService *vectorservice.VectorSearchService,
	embeddingService *vectorservice.VectorEmbeddingService,
	vectorRepo repository.VectorRepository,
	vectorTaskRepo repository.VectorTaskRepository,
	repoRepo repository.RepoRepository,
	docRepo repository.DocumentRepository,
) *VectorHandler {
	return &VectorHandler{
		searchService:    searchService,
		embeddingService: embeddingService,
		vectorRepo:       vectorRepo,
		vectorTaskRepo:   vectorTaskRepo,
		repoRepo:         repoRepo,
		docRepo:          docRepo,
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query         string                 `json:"query" binding:"required"`
	Model         string                 `json:"model"`
	RepositoryID  uint                   `json:"repository_id"`
	TopK          int                    `json:"top_k"`
	MinSimilarity float64                `json:"min_similarity"`
	Filters       map[string]interface{} `json:"filters"`
}

// Search 语义搜索
// POST /api/vectors/search
func (h *VectorHandler) Search(c *gin.Context) {
	if h.searchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector search service not configured"})
		return
	}

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if req.MinSimilarity <= 0 {
		req.MinSimilarity = 0.5
	}

	results, err := h.searchService.Search(c.Request.Context(), req.Query, req.TopK, req.MinSimilarity, req.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// FindSimilarDocuments 查找相似文档
// GET /api/documents/:id/similar
func (h *VectorHandler) FindSimilarDocuments(c *gin.Context) {
	if h.searchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "vector search service not configured"})
		return
	}

	docID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	topK := 5
	if k := c.Query("top_k"); k != "" {
		if n, err := strconv.Atoi(k); err == nil && n > 0 {
			topK = n
		}
	}

	minSimilarity := 0.7
	if s := c.Query("min_similarity"); s != "" {
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			minSimilarity = n
		}
	}

	results, err := h.searchService.FindSimilarDocuments(c.Request.Context(), uint(docID), topK, minSimilarity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// GenerateVector 为文档生成向量
// POST /api/documents/:id/vector/generate
func (h *VectorHandler) GenerateVector(c *gin.Context) {
	if h.embeddingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "embedding service not configured"})
		return
	}

	docID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	if err := h.embeddingService.GenerateForDocument(c.Request.Context(), uint(docID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vector generation task created"})
}

// GenerateRepositoryVectors 批量为仓库生成向量
// POST /api/repositories/:id/vectors/generate
func (h *VectorHandler) GenerateRepositoryVectors(c *gin.Context) {
	if h.embeddingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "embedding service not configured"})
		return
	}

	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	if err := h.embeddingService.GenerateForRepository(c.Request.Context(), uint(repoID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vector generation tasks created"})
}

// RegenerateVector 重新生成文档的向量
// POST /api/documents/:id/vector/regenerate
func (h *VectorHandler) RegenerateVector(c *gin.Context) {
	if h.embeddingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "embedding service not configured"})
		return
	}

	docID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	if err := h.embeddingService.RegenerateForDocument(c.Request.Context(), uint(docID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vector regeneration task created"})
}

// GetVectorStatus 获取向量生成状态
// GET /api/vectors/status
func (h *VectorHandler) GetVectorStatus(c *gin.Context) {
	status, err := h.vectorRepo.GetStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// DeleteVector 删除文档的向量
// DELETE /api/documents/:id/vector
func (h *VectorHandler) DeleteVector(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	if err := h.vectorRepo.DeleteByDocumentID(c.Request.Context(), uint(docID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vector deleted"})
}

// RepositoryVectorStatus 仓库向量化状态
type RepositoryVectorStatus struct {
	RepositoryID      uint   `json:"repository_id"`
	RepositoryName    string `json:"repository_name"`
	TotalDocuments    int64  `json:"total_documents"`
	VectorizedCount   int64  `json:"vectorized_count"`
	Status            string `json:"status"` // not_started, partial, completed
}

// GetRepositoryVectorStatusList 获取所有仓库的向量化状态
// GET /api/vectors/repositories/status
func (h *VectorHandler) GetRepositoryVectorStatusList(c *gin.Context) {
	// 获取所有仓库
	repos, err := h.repoRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取所有已向量化的文档 ID
	vectorizedDocs, err := h.vectorRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建向量化的文档集合
	vectorizedSet := make(map[uint]bool)
	for _, v := range vectorizedDocs {
		vectorizedSet[v.DocumentID] = true
	}

	// 计算每个仓库的向量化状态
	result := make([]RepositoryVectorStatus, 0, len(repos))
	for _, repo := range repos {
		// 获取仓库的所有文档
		docs, err := h.docRepo.GetByRepository(repo.ID)
		if err != nil {
			continue
		}

		var vectorizedCount int64
		for _, doc := range docs {
			if vectorizedSet[doc.ID] {
				vectorizedCount++
			}
		}

		totalDocs := int64(len(docs))
		status := "not_started"
		if vectorizedCount > 0 && vectorizedCount < totalDocs {
			status = "partial"
		} else if vectorizedCount == totalDocs && totalDocs > 0 {
			status = "completed"
		}

		result = append(result, RepositoryVectorStatus{
			RepositoryID:    repo.ID,
			RepositoryName:  repo.Name,
			TotalDocuments:  totalDocs,
			VectorizedCount: vectorizedCount,
			Status:          status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"list":  result,
		"total": len(result),
	})
}

// VectorTaskDTO 向量任务数据传输对象
type VectorTaskDTO struct {
	ID             uint   `json:"id"`
	DocumentID     uint   `json:"document_id"`
	DocumentTitle  string `json:"document_title"`
	RepositoryID   uint   `json:"repository_id"`
	RepositoryName string `json:"repository_name"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message"`
	CreatedAt      string `json:"created_at"`
	StartedAt      string `json:"started_at"`
	CompletedAt    string `json:"completed_at"`
}

// GetVectorTasks 获取向量任务列表
// GET /api/vectors/tasks
func (h *VectorHandler) GetVectorTasks(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取任务列表
	var tasks []model.VectorTask
	if status != "" {
		// 按状态过滤
		allTasks, err := h.vectorTaskRepo.GetPendingTasks(c.Request.Context(), 1000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, t := range allTasks {
			if t.Status == status {
				tasks = append(tasks, t)
			}
		}
	} else {
		// 获取所有任务（需要实现新方法或使用现有方法）
		// 暂时获取 pending 任务
		var err error
		tasks, err = h.vectorTaskRepo.GetPendingTasks(c.Request.Context(), 1000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 转换为 DTO
	dtos := make([]VectorTaskDTO, 0, len(tasks))
	for _, t := range tasks {
		dto := VectorTaskDTO{
			ID:           t.ID,
			DocumentID:   t.DocumentID,
			Status:       t.Status,
			ErrorMessage: t.ErrorMessage,
		}
		if t.CreatedAt.IsZero() {
			dto.CreatedAt = ""
		} else {
			dto.CreatedAt = t.CreatedAt.Format("2006-01-02 15:04:05")
		}
		if t.StartedAt != nil {
			dto.StartedAt = t.StartedAt.Format("2006-01-02 15:04:05")
		}
		if t.CompletedAt != nil {
			dto.CompletedAt = t.CompletedAt.Format("2006-01-02 15:04:05")
		}

		// 获取文档信息
		doc, err := h.docRepo.Get(t.DocumentID)
		if err == nil {
			dto.DocumentTitle = doc.Title
			dto.RepositoryID = doc.RepositoryID

			// 获取仓库名称
			repo, err := h.repoRepo.Get(doc.RepositoryID)
			if err == nil {
				dto.RepositoryName = repo.Name
			}
		}

		dtos = append(dtos, dto)
	}

	// 分页
	total := len(dtos)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		dtos = []VectorTaskDTO{}
	} else if end > total {
		dtos = dtos[start:]
	} else {
		dtos = dtos[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"list":      dtos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// RegisterRoutes 注册路由
func (h *VectorHandler) RegisterRoutes(r *gin.RouterGroup) {
	vectors := r.Group("/vectors")
	{
		vectors.GET("/status", h.GetVectorStatus)
		vectors.POST("/search", h.Search)
		vectors.GET("/repositories/status", h.GetRepositoryVectorStatusList)
		vectors.GET("/tasks", h.GetVectorTasks)
	}

	// 文档向量操作
	r.POST("/documents/:id/vector/generate", h.GenerateVector)
	r.POST("/documents/:id/vector/regenerate", h.RegenerateVector)
	r.DELETE("/documents/:id/vector", h.DeleteVector)
	r.GET("/documents/:id/similar", h.FindSimilarDocuments)

	// 仓库向量操作
	r.POST("/repositories/:id/vectors/generate", h.GenerateRepositoryVectors)
}
