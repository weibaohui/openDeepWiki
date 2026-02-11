package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

type RepositoryHandler struct {
	service     *service.RepositoryService
	taskService *service.TaskService
}

func NewRepositoryHandler(service *service.RepositoryService, taskService *service.TaskService) *RepositoryHandler {
	return &RepositoryHandler{
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
	task, err := h.taskService.CreateTocWriteTask(ctx, uint(id), "目录分析", 10)
	if err != nil {
		klog.Errorf("AnalyzeDirectory failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "directory analysis started",
		"task":    task,
	})
}

func (h *RepositoryHandler) AnalyzeDatabaseModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	ctx := context.Background()
	task, err := h.taskService.CreateDocWriteTask(ctx, uint(id), "数据库模型分析", 20, domain.DBModelWriter)
	if err != nil {
		klog.Errorf("AnalyzeDatabaseModel failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "database model analysis started",
		"task":    task,
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
	task, err := h.taskService.CreateDocWriteTask(ctx, uint(id), "API接口分析", 20, domain.APIWriter)
	if err != nil {
		klog.Errorf("CreateDocWriteTask failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "api analysis started",
		"task":    task,
	})
}

type AnalyzeProblemRequest struct {
	Content string `json:"content" binding:"required"`
}

// AnalyzeUserRequest 处理问题分析的触发请求。
func (h *RepositoryHandler) AnalyzeUserRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req AnalyzeProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem is required"})
		return
	}

	ctx := context.Background()
	task, err := h.taskService.CreateUserRequestTask(ctx, uint(id), req.Content, 30)
	if err != nil {
		klog.Errorf("CreateUserRequestTask failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "problem analysis started",
		"task":    task,
	})
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

	if err := h.service.CloneRepository(uint(id)); err != nil {
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
