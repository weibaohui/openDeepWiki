package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

// APIKeyHandler API Key 处理器
type APIKeyHandler struct {
	service service.APIKeyService
}

// NewAPIKeyHandler 创建 API Key 处理器
func NewAPIKeyHandler(service service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *APIKeyHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/api-keys", h.ListAPIKeys)
	router.POST("/api-keys", h.CreateAPIKey)
	router.GET("/api-keys/:id", h.GetAPIKey)
	router.PUT("/api-keys/:id", h.UpdateAPIKey)
	router.DELETE("/api-keys/:id", h.DeleteAPIKey)
	router.PATCH("/api-keys/:id/status", h.UpdateStatus)
	router.GET("/api-keys/stats", h.GetStats)
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name     string `json:"name" binding:"required"`
	Provider string `json:"provider" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
	Model    string `json:"model" binding:"required"`
	Priority int    `json:"priority"`
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"` // enabled/disabled
}

// APIKeyResponse API Key 响应（脱敏）
type APIKeyResponse struct {
	ID               uint       `json:"id"`
	Name             string     `json:"name"`
	Provider         string     `json:"provider"`
	BaseURL          string     `json:"base_url"`
	APIKey           string     `json:"api_key"`       // 脱敏后
	Model            string     `json:"model"`
	Priority         int        `json:"priority"`
	Status           string     `json:"status"`
	RequestCount     int        `json:"request_count"`
	ErrorCount       int        `json:"error_count"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// CreateAPIKey 创建 API Key 配置
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.V(6).Infof("CreateAPIKey: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.service.CreateAPIKey(c.Request.Context(), &service.CreateAPIKeyRequest{
		Name:     req.Name,
		Provider: req.Provider,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Model:    req.Model,
		Priority: req.Priority,
	})
	if err != nil {
		klog.Errorf("CreateAPIKey: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.toResponse(apiKey))
}

// GetAPIKey 获取 API Key 配置
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	id := c.Param("id")
	var apiKeyID uint
	if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil || apiKeyID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	apiKey, err := h.service.GetAPIKey(c.Request.Context(), apiKeyID)
	if err != nil {
		klog.Errorf("GetAPIKey: failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.toResponse(apiKey))
}

// ListAPIKeys 列出所有 API Key 配置
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	apiKeys, err := h.service.ListAPIKeys(c.Request.Context())
	if err != nil {
		klog.Errorf("ListAPIKeys: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]*APIKeyResponse, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		responses = append(responses, h.toResponse(apiKey))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": len(responses),
	})
}

// UpdateAPIKey 更新 API Key 配置
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	var apiKeyID uint
	if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil || apiKeyID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.V(6).Infof("UpdateAPIKey: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.service.UpdateAPIKey(c.Request.Context(), apiKeyID, &service.UpdateAPIKeyRequest{
		Name:     req.Name,
		Provider: req.Provider,
		BaseURL:  req.BaseURL,
		APIKey:   req.APIKey,
		Model:    req.Model,
		Priority: req.Priority,
	})
	if err != nil {
		klog.Errorf("UpdateAPIKey: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.toResponse(apiKey))
}

// DeleteAPIKey 删除 API Key 配置
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	var apiKeyID uint
	if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil || apiKeyID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.DeleteAPIKey(c.Request.Context(), apiKeyID); err != nil {
		klog.Errorf("DeleteAPIKey: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// UpdateStatus 更新状态
func (h *APIKeyHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var apiKeyID uint
	if _, err := fmt.Sscanf(id, "%d", &apiKeyID); err != nil || apiKeyID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.V(6).Infof("UpdateStatus: invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateAPIKeyStatus(c.Request.Context(), apiKeyID, req.Status); err != nil {
		klog.Errorf("UpdateStatus: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}

// GetStats 获取统计信息
func (h *APIKeyHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		klog.Errorf("GetStats: failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// toResponse 转换为响应对象（脱敏 API Key）
func (h *APIKeyHandler) toResponse(apiKey *model.APIKey) *APIKeyResponse {
	return &APIKeyResponse{
		ID:               apiKey.ID,
		Name:             apiKey.Name,
		Provider:         apiKey.Provider,
		BaseURL:          apiKey.BaseURL,
		APIKey:           apiKey.MaskAPIKey(),
		Model:            apiKey.Model,
		Priority:         apiKey.Priority,
		Status:           apiKey.Status,
		RequestCount:     apiKey.RequestCount,
		ErrorCount:       apiKey.ErrorCount,
		LastUsedAt:       apiKey.LastUsedAt,
		RateLimitResetAt: apiKey.RateLimitResetAt,
		CreatedAt:        apiKey.CreatedAt,
		UpdatedAt:        apiKey.UpdatedAt,
	}
}
