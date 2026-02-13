package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type DocumentHandler struct {
	bus     *eventbus.DocEventBus
	service *service.DocumentService
}

// NewDocumentHandler 创建文档处理器
func NewDocumentHandler(bus *eventbus.DocEventBus, service *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{
		service: service,
		bus:     bus,
	}
}

// GetByRepository 获取仓库下文档列表
func (h *DocumentHandler) GetByRepository(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	docs, err := h.service.GetByRepository(uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// Get 获取单个文档详情
func (h *DocumentHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	doc, err := h.service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// GetVersions 获取文档版本列表
func (h *DocumentHandler) GetVersions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	docs, err := h.service.GetVersions(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// Update 更新文档内容
func (h *DocumentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.service.Update(uint(id), req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// Export 导出仓库下所有文档
func (h *DocumentHandler) Export(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	data, filename, err := h.service.ExportAll(uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/zip", data)
}

// Redirect 重定向到原始代码文件
func (h *DocumentHandler) Redirect(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	redirectURL, err := h.service.GetRedirectURL(uint(id), path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, redirectURL)
}

// GetIndex 获取仓库文档索引内容
func (h *DocumentHandler) GetIndex(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	content, err := h.service.GetIndex(uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

// SubmitRating 提交文档评分
func (h *DocumentHandler) SubmitRating(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Score int `json:"score" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Score < 1 || req.Score > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score must be between 1 and 5"})
		return
	}

	stats, err := h.service.SubmitRating(uint(id), req.Score)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 发布文档评分事件
	h.bus.Publish(c, eventbus.DocEventRated, eventbus.DocEvent{
		Type:   eventbus.DocEventRated,
		DocID:  uint(id),
		Rating: req.Score,
	})

	c.JSON(http.StatusOK, stats)
}

// GetRatingStats 获取文档评分统计
func (h *DocumentHandler) GetRatingStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	stats, err := h.service.GetRatingStats(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetTokenUsage 获取文档的 Token 用量数据
func (h *DocumentHandler) GetTokenUsage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid document id"})
		return
	}

	usage, err := h.service.GetTokenUsage(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "failed to get token usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    usage,
	})
}
