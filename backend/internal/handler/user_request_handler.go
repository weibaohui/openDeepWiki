package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

// UserRequestHandler 用户需求处理器
type UserRequestHandler struct {
	service     service.UserRequestService
	taskBus     *eventbus.TaskEventBus
	taskService *service.TaskService
}

// NewUserRequestHandler 创建用户需求处理器实例
func NewUserRequestHandler(service service.UserRequestService, taskBus *eventbus.TaskEventBus, taskService *service.TaskService) *UserRequestHandler {
	klog.V(6).Infof("[handler] 创建 UserRequestHandler")
	return &UserRequestHandler{
		service:     service,
		taskBus:     taskBus,
		taskService: taskService,
	}
}

// CreateUserRequest 创建用户需求
// POST /api/repositories/:id/user-requests
func (h *UserRequestHandler) CreateUserRequest(c *gin.Context) {
	klog.V(6).Infof("[handler] 创建用户需求请求")

	// 解析仓库 ID
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		klog.Warningf("[handler] 创建用户需求失败: 无效的仓库ID, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid repository id"})
		return
	}

	// 解析请求体
	var req struct {
		Content string `json:"content" binding:"required,max=200"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Warningf("[handler] 创建用户需求失败: 无效的请求体, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "content is required and max length is 200"})
		return
	}

	// 创建用户需求
	request, err := h.service.CreateRequest(uint(repoID), req.Content)
	if err != nil {
		klog.Errorf("[handler] 创建用户需求失败: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	klog.V(6).Infof("[handler] 创建用户需求成功: id=%d", request.ID)

	// 创建用户需求成功后，触发分析任务
	ctx := context.Background()
	h.taskBus.Publish(ctx, eventbus.TaskEventUserRequest, eventbus.TaskEvent{
		Type:         eventbus.TaskEventUserRequest,
		RepositoryID: uint(repoID),
		Title:        req.Content,
		Outline:      req.Content,
		SortOrder:    30,
		WriterName:   domain.UserRequestWriter,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "需求已提交",
		"data":    request,
	})
}

// ListUserRequests 获取用户需求列表
// GET /api/repositories/:id/user-requests
func (h *UserRequestHandler) ListUserRequests(c *gin.Context) {
	klog.V(6).Infof("[handler] 获取用户需求列表请求")

	// 解析仓库 ID
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		klog.Warningf("[handler] 获取用户需求列表失败: 无效的仓库ID, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid repository id"})
		return
	}

	// 解析分页参数
	page := 1
	pageSize := 20
	status := c.Query("status")

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 50 {
			pageSize = s
		}
	}

	// 获取列表
	requests, total, err := h.service.ListRequests(uint(repoID), page, pageSize, status)
	if err != nil {
		klog.Errorf("[handler] 获取用户需求列表失败: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	klog.V(6).Infof("[handler] 获取用户需求列表成功: total=%d, returned=%d", total, len(requests))
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      requests,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetUserRequest 获取用户需求详情
// GET /api/user-requests/:id
func (h *UserRequestHandler) GetUserRequest(c *gin.Context) {
	klog.V(6).Infof("[handler] 获取用户需求详情请求")

	// 解析需求 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		klog.Warningf("[handler] 获取用户需求详情失败: 无效的ID, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	// 获取详情
	request, err := h.service.GetRequest(uint(id))
	if err != nil {
		klog.Errorf("[handler] 获取用户需求详情失败: error=%v", err)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "request not found"})
		return
	}

	klog.V(6).Infof("[handler] 获取用户需求详情成功: id=%d", id)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    request,
	})
}

// DeleteUserRequest 删除用户需求
// DELETE /api/user-requests/:id
func (h *UserRequestHandler) DeleteUserRequest(c *gin.Context) {
	klog.V(6).Infof("[handler] 删除用户需求请求")

	// 解析需求 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		klog.Warningf("[handler] 删除用户需求失败: 无效的ID, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	// 删除需求
	if err := h.service.DeleteRequest(uint(id)); err != nil {
		klog.Errorf("[handler] 删除用户需求失败: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	klog.V(6).Infof("[handler] 删除用户需求成功: id=%d", id)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data":    nil,
	})
}

// UpdateUserRequestStatus 更新用户需求状态
// PATCH /api/user-requests/:id/status
func (h *UserRequestHandler) UpdateUserRequestStatus(c *gin.Context) {
	klog.V(6).Infof("[handler] 更新用户需求状态请求")

	// 解析需求 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		klog.Warningf("[handler] 更新用户需求状态失败: 无效的ID, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	// 解析请求体
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Warningf("[handler] 更新用户需求状态失败: 无效的请求体, error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "status is required"})
		return
	}

	// 更新状态
	if err := h.service.UpdateStatus(uint(id), req.Status); err != nil {
		klog.Errorf("[handler] 更新用户需求状态失败: error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	klog.V(6).Infof("[handler] 更新用户需求状态成功: id=%d, status=%s", id, req.Status)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "状态更新成功",
		"data":    nil,
	})
}
