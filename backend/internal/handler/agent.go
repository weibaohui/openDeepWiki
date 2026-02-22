package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

// AgentHandler Agent 处理器
type AgentHandler struct {
	service service.AgentServiceAgentService
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(agentService service.AgentServiceAgentService) *AgentHandler {
	return &AgentHandler{service: agentService}
}

// RegisterRoutes 注册路由
func (h *AgentHandler) RegisterRoutes(router *gin.RouterGroup) {
	agents := router.Group("/agents")
	{
		agents.GET("", h.ListAgents)
		agents.GET("/:filename", h.GetAgent)
		agents.PUT("/:filename", h.SaveAgent)
		agents.GET("/:filename/versions", h.GetVersions)
		agents.GET("/:filename/versions/:version", h.GetVersionContent)
		agents.POST("/:filename/versions/:version/restore", h.RestoreVersion)
		agents.DELETE("/:filename/versions/:version", h.DeleteVersion)
		agents.DELETE("/:filename/versions", h.DeleteVersions)
	}
}

// ListAgents 列出所有 Agent
func (h *AgentHandler) ListAgents(c *gin.Context) {
	agents, err := h.service.ListAgents(c.Request.Context())
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to list agents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  agents,
		"total": len(agents),
	})
}

// GetAgent 获取指定 Agent 的内容
func (h *AgentHandler) GetAgent(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	agent, err := h.service.GetAgent(c.Request.Context(), fileName)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to get agent: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// SaveAgentRequest 保存 Agent 请求
type SaveAgentRequest struct {
	Content string `json:"content" binding:"required"`
}

// SaveAgent 保存 Agent 定义
func (h *AgentHandler) SaveAgent(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	var req SaveAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.V(6).Infof("[AgentHandler] Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.SaveAgent(c.Request.Context(), fileName, req.Content, "web", nil)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to save agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetVersions 获取 Agent 的版本历史
func (h *AgentHandler) GetVersions(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	versions, err := h.service.GetVersions(c.Request.Context(), fileName)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to get versions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_name": fileName,
		"versions":  versions,
	})
}

// GetVersionContent 获取指定版本的完整内容
func (h *AgentHandler) GetVersionContent(c *gin.Context) {
	fileName := c.Param("filename")
	versionParam := c.Param("version")

	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil || version <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	content, err := h.service.GetVersionContent(c.Request.Context(), fileName, version)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to get version content: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

// RestoreVersion 从历史版本恢复 Agent
func (h *AgentHandler) RestoreVersion(c *gin.Context) {
	fileName := c.Param("filename")
	versionParam := c.Param("version")

	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil || version <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	result, err := h.service.RestoreVersion(c.Request.Context(), fileName, version)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to restore version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteVersionsRequest 批量删除请求
type DeleteVersionsRequest struct {
	Versions []int `json:"versions" binding:"required"`
}

// DeleteVersion 删除指定历史版本
func (h *AgentHandler) DeleteVersion(c *gin.Context) {
	fileName := c.Param("filename")
	versionParam := c.Param("version")

	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil || version <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	err := h.service.DeleteVersion(c.Request.Context(), fileName, version)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to delete version: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// DeleteVersions 批量删除历史版本
func (h *AgentHandler) DeleteVersions(c *gin.Context) {
	fileName := c.Param("filename")

	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	// 防止目录遍历攻击
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	var req DeleteVersionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.V(6).Infof("[AgentHandler] Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Versions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no versions to delete"})
		return
	}

	err := h.service.DeleteVersions(c.Request.Context(), fileName, req.Versions)
	if err != nil {
		klog.Errorf("[AgentHandler] Failed to delete versions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": len(req.Versions),
	})
}
