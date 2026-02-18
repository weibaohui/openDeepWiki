package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type RepositoryHandler struct {
	repoBus     *eventbus.RepositoryEventBus
	taskBus     *eventbus.TaskEventBus
	service     *service.RepositoryService
	taskService *service.TaskService
}

func NewRepositoryHandler(repoBus *eventbus.RepositoryEventBus, taskBus *eventbus.TaskEventBus, service *service.RepositoryService, taskService *service.TaskService) *RepositoryHandler {
	return &RepositoryHandler{
		repoBus:     repoBus,
		taskBus:     taskBus,
		service:     service,
		taskService: taskService,
	}
}

func (h *RepositoryHandler) Create(c *gin.Context) {
	var req service.CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRepositoryURL):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrRepositoryAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	ctx := context.Background()
	h.repoBus.Publish(ctx, eventbus.RepositoryEventAdded, eventbus.RepositoryEvent{
		Type:         eventbus.RepositoryEventAdded,
		RepositoryID: repo.ID,
	})

	c.JSON(http.StatusCreated, repo)
}

func (h *RepositoryHandler) List(c *gin.Context) {
	repos, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, repos)
}

func (h *RepositoryHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	repo, err := h.service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	c.JSON(http.StatusOK, repo)
}

func (h *RepositoryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDeleteRepoInvalidStatus):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *RepositoryHandler) RunAllTasks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.RunAllTasks(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tasks started"})
}

func (h *RepositoryHandler) AnalyzeDirectory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()

	h.taskBus.Publish(ctx, eventbus.TaskEventTocWrite, eventbus.TaskEvent{
		Type:         eventbus.TaskEventTocWrite,
		RepositoryID: uint(id),
		Title:        "目录分析",
		SortOrder:    10,
		WriterName:   domain.TocWriter,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "directory analysis started",
	})

}

func (h *RepositoryHandler) AnalyzeDatabaseModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	h.taskBus.Publish(ctx, eventbus.TaskEventDocWrite, eventbus.TaskEvent{
		Type:         eventbus.TaskEventDocWrite,
		RepositoryID: uint(id),
		Title:        "数据库模型分析",
		SortOrder:    20,
		WriterName:   domain.DBModelWriter,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "database model analysis started",
	})
}

// AnalyzeAPI 处理API接口分析的触发请求。
func (h *RepositoryHandler) AnalyzeAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	h.taskBus.Publish(ctx, eventbus.TaskEventDocWrite, eventbus.TaskEvent{
		Type:         eventbus.TaskEventDocWrite,
		RepositoryID: uint(id),
		Title:        "API接口分析",
		SortOrder:    20,
		WriterName:   domain.APIWriter,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "api analysis started",
	})
}

// IncrementalAnalysis 处理增量分析的触发请求。
func (h *RepositoryHandler) IncrementalAnalysis(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	h.taskBus.Publish(ctx, eventbus.TaskEventIncrementalWrite, eventbus.TaskEvent{
		Type:         eventbus.TaskEventIncrementalWrite,
		RepositoryID: uint(id),
		Title:        "增量分析",
		SortOrder:    20,
		WriterName:   domain.IncrementalWriter,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "incremental analysis started",
	})
}

type AnalyzeProblemRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *RepositoryHandler) SetReady(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.SetReady(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "仓库状态已设置为就绪"})
}

// Clone 重新下载仓库（删除本地目录并重新克隆）
// 仅在非 cloning/analyzing 状态下允许触发
func (h *RepositoryHandler) Clone(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	if err := h.service.CloneRepository(ctx, uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "clone started"})
}

func (h *RepositoryHandler) PurgeLocal(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.PurgeLocalDir(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "local directory purged"})
}

// GetIncrementalHistory 获取仓库的增量同步历史记录。
func (h *RepositoryHandler) GetIncrementalHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 解析可选的 limit 参数
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := h.service.GetIncrementalHistory(uint(id), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}
