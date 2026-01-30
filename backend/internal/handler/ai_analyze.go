package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/opendeepwiki/backend/internal/service"
)

// AIAnalyzeHandler AI分析Handler
type AIAnalyzeHandler struct {
	service *service.AIAnalyzeService
}

// NewAIAnalyzeHandler 创建Handler
func NewAIAnalyzeHandler(service *service.AIAnalyzeService) *AIAnalyzeHandler {
	return &AIAnalyzeHandler{
		service: service,
	}
}

// StartAnalysis 启动AI分析
// POST /api/repositories/:id/ai-analyze
func (h *AIAnalyzeHandler) StartAnalysis(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	resp, err := h.service.StartAnalysis(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetAnalysisStatus 获取AI分析状态
// GET /api/repositories/:id/ai-analysis-status
func (h *AIAnalyzeHandler) GetAnalysisStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	task, err := h.service.GetAnalysisStatus(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetAnalysisResult 获取AI分析结果
// GET /api/repositories/:id/ai-analysis-result
func (h *AIAnalyzeHandler) GetAnalysisResult(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	content, err := h.service.GetAnalysisResult(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": content,
	})
}
