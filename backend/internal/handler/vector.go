package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	vectorservice "github.com/weibaohui/opendeepwiki/backend/internal/service/vector"
)

// VectorHandler 向量搜索处理器
type VectorHandler struct {
	searchService    *vectorservice.VectorSearchService
	embeddingService *vectorservice.VectorEmbeddingService
}

// NewVectorHandler 创建向量处理器
func NewVectorHandler(
	searchService *vectorservice.VectorSearchService,
	embeddingService *vectorservice.VectorEmbeddingService,
) *VectorHandler {
	return &VectorHandler{
		searchService:    searchService,
		embeddingService: embeddingService,
	}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query          string                 `json:"query" binding:"required"`
	Model          string                 `json:"model"`
	RepositoryID   uint                   `json:"repository_id"`
	TopK           int                    `json:"top_k"`
	MinSimilarity  float64                `json:"min_similarity"`
	Filters        map[string]interface{} `json:"filters"`
}

// Search 语义搜索
// POST /api/vectors/search
func (h *VectorHandler) Search(c *gin.Context) {
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
	status, err := h.embeddingService.GetStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// DeleteVector 删除文档的向量
// DELETE /api/documents/:id/vector
func (h *VectorHandler) DeleteVector(c *gin.Context) {
	// 这里需要从 handler 访问 repository，暂时返回未实现
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
}