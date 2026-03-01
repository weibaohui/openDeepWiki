package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type EmbeddingKeyHandler struct {
	service *service.EmbeddingKeyService
}

func NewEmbeddingKeyHandler(service *service.EmbeddingKeyService) *EmbeddingKeyHandler {
	return &EmbeddingKeyHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *EmbeddingKeyHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/embedding-keys", h.List)
	router.POST("/embedding-keys", h.Create)
	router.GET("/embedding-keys/:id", h.GetByID)
	router.PUT("/embedding-keys/:id", h.Update)
	router.DELETE("/embedding-keys/:id", h.Delete)
	router.POST("/embedding-keys/:id/enable", h.Enable)
	router.POST("/embedding-keys/:id/disable", h.Disable)
	router.POST("/embedding-keys/:id/test", h.TestConnection)
	router.GET("/embedding-keys/stats", h.GetUsageStats)
}

// List 列出所有嵌入模型配置
// GET /api/embedding-keys
func (h *EmbeddingKeyHandler) List(c *gin.Context) {
	keys, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, keys)
}

// GetByID 根据ID获取配置
// GET /api/embedding-keys/:id
func (h *EmbeddingKeyHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	key, err := h.service.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, key)
}

// Create 创建嵌入模型配置
// POST /api/embedding-keys
func (h *EmbeddingKeyHandler) Create(c *gin.Context) {
	var key model.EmbeddingKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Create(c.Request.Context(), &key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, key)
}

// Update 更新配置
// PUT /api/embedding-keys/:id
func (h *EmbeddingKeyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var key model.EmbeddingKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key.ID = uint(id)
	if err := h.service.Update(c.Request.Context(), &key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, key)
}

// Delete 删除配置
// DELETE /api/embedding-keys/:id
func (h *EmbeddingKeyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Enable 启用配置
// POST /api/embedding-keys/:id/enable
func (h *EmbeddingKeyHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Enable(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "enabled"})
}

// Disable 禁用配置
// POST /api/embedding-keys/:id/disable
func (h *EmbeddingKeyHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Disable(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disabled"})
}

// TestConnection 测试连接
// POST /api/embedding-keys/:id/test
func (h *EmbeddingKeyHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.TestConnection(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetUsageStats 获取使用统计
// GET /api/embedding-keys/stats
func (h *EmbeddingKeyHandler) GetUsageStats(c *gin.Context) {
	stats, err := h.service.GetUsageStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}