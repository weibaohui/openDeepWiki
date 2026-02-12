package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type TaskHandler struct {
	service         *service.TaskService
	taskUsageService service.TaskUsageService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{
		service:         service,
		taskUsageService: service.GetTaskUsageService(),
	}
}

// GetByRepository 获取仓库的所有任务
func (h *TaskHandler) GetByRepository(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	tasks, err := h.service.GetByRepository(uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetStats 获取仓库的任务统计信息
func (h *TaskHandler) GetStats(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return
	}

	stats, err := h.service.GetTaskStats(uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Enqueue 提交任务到队列（替代原来的Run方法）
// 接口变更：从"立即执行"改为"提交作业"
func (h *TaskHandler) Enqueue(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取任务信息
	_, err = h.service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 提交任务到编排器队列
	if err := h.service.Enqueue(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task enqueued",
		"status":  "queued",
	})
}

// Run 兼容旧接口（保持向后兼容）
// 内部调用Enqueue方法
func (h *TaskHandler) Run(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取任务信息
	_, err = h.service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// 提交任务到编排器队列
	if err := h.service.Enqueue(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task started",
		"status":  "queued",
	})
}

// Get 获取单个任务详情
func (h *TaskHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := h.service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// Reset 重置任务
func (h *TaskHandler) Reset(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.Reset(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task reset",
		"status":  "pending",
	})
}

// ForceReset 强制重置任务，无论当前状态
func (h *TaskHandler) ForceReset(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.ForceReset(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task force reset",
		"status":  "pending",
	})
}

// Retry 重试任务（Reset + Enqueue）
func (h *TaskHandler) Retry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.Retry(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task retry started",
		"status":  "queued",
	})
}

// ReGenByNewTask 重新生成任务
func (h *TaskHandler) ReGenByNewTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.ReGenByNewTask(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task re-generated",
		"status":  "queued",
	})
}

// Cancel 取消任务
func (h *TaskHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.Cancel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task canceled",
		"status":  "canceled",
	})
}

// CleanupStuck 清理超时的卡住任务
func (h *TaskHandler) CleanupStuck(c *gin.Context) {
	// 默认超时时间为 10 分钟
	timeout := 10 * time.Minute
	if t := c.Query("timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	affected, err := h.service.CleanupStuckTasks(timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "cleanup completed",
		"affected": affected,
		"timeout":  timeout.String(),
	})
}

// GetStuck 获取卡住的任务列表
func (h *TaskHandler) GetStuck(c *gin.Context) {
	// 默认超时时间为 10 分钟
	timeout := 10 * time.Minute
	if t := c.Query("timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	tasks, err := h.service.GetStuckTasks(timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":   tasks,
		"count":   len(tasks),
		"timeout": timeout.String(),
	})
}

// GetOrchestratorStatus 获取编排器状态
// 新增接口，用于监控任务队列和执行状态
func (h *TaskHandler) GetOrchestratorStatus(c *gin.Context) {
	status := h.service.GetOrchestratorStatus()
	if status == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "orchestrator not available",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Delete 删除任务（删除单个任务）
func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "task deleted",
	})
}

// Monitor 全局任务监控
func (h *TaskHandler) Monitor(c *gin.Context) {
	data, err := h.service.GetGlobalMonitorData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetTaskUsage 获取任务 Token 用量
func (h *TaskHandler) GetTaskUsage(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid task id"})
		return
	}

	usage, err := h.taskUsageService.GetByTaskID(c.Request.Context(), uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "failed to get task usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    usage,
	})
}
