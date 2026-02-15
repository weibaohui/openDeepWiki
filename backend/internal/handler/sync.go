package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	syncservice "github.com/weibaohui/opendeepwiki/backend/internal/service/sync"
	"k8s.io/klog/v2"
)

type SyncHandler struct {
	service *syncservice.Service
}

func NewSyncHandler(service *syncservice.Service) *SyncHandler {
	return &SyncHandler{service: service}
}

func (h *SyncHandler) RegisterRoutes(router *gin.RouterGroup) {
	syncGroup := router.Group("/sync")
	{
		syncGroup.GET("/ping", h.Ping)
		syncGroup.POST("", h.Start)
		syncGroup.POST("/pull", h.Pull)
		syncGroup.GET("/status/:sync_id", h.Status)
		syncGroup.GET("/repository-list", h.RepositoryList)
		syncGroup.GET("/document-list", h.DocumentList)
		syncGroup.GET("/target-list", h.TargetList)
		syncGroup.POST("/target-save", h.TargetSave)
		syncGroup.POST("/target-delete", h.TargetDelete)
		syncGroup.POST("/pull-export", h.PullExport)
		syncGroup.POST("/repository-upsert", h.RepositoryUpsert)
		syncGroup.POST("/repository-clear", h.RepositoryClear)
		syncGroup.POST("/task-create", h.TaskCreate)
		syncGroup.POST("/document-create", h.DocumentCreate)
		syncGroup.POST("/task-update-docid", h.TaskUpdateDocID)
		syncGroup.POST("/task-usage-create", h.TaskUsageCreate)
	}
}

func (h *SyncHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, syncdto.PingResponse{Code: "OK"})
}

func (h *SyncHandler) Start(c *gin.Context) {
	var req syncdto.StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.service.Start(c.Request.Context(), req.TargetServer, req.RepositoryID, req.DocumentIDs, req.ClearTarget)
	if err != nil {
		klog.Errorf("[sync.Start] 启动同步失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.StartResponse{
		Code: "OK",
		Data: syncdto.StartData{
			SyncID:       status.SyncID,
			RepositoryID: status.RepositoryID,
			TotalTasks:   status.TotalTasks,
			Status:       status.Status,
		},
	})
}

func (h *SyncHandler) Pull(c *gin.Context) {
	var req syncdto.PullStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.service.StartPull(c.Request.Context(), req.TargetServer, req.RepositoryID, req.DocumentIDs, req.ClearLocal)
	if err != nil {
		klog.Errorf("[sync.Pull] 启动拉取失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.StartResponse{
		Code: "OK",
		Data: syncdto.StartData{
			SyncID:       status.SyncID,
			RepositoryID: status.RepositoryID,
			TotalTasks:   status.TotalTasks,
			Status:       status.Status,
		},
	})
}

func (h *SyncHandler) Status(c *gin.Context) {
	syncID := c.Param("sync_id")
	status, ok := h.service.GetStatus(syncID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "sync not found"})
		return
	}

	c.JSON(http.StatusOK, syncdto.StatusResponse{
		Code: "OK",
		Data: syncdto.StatusData{
			SyncID:         status.SyncID,
			RepositoryID:   status.RepositoryID,
			TotalTasks:     status.TotalTasks,
			CompletedTasks: status.CompletedTasks,
			FailedTasks:    status.FailedTasks,
			Status:         status.Status,
			CurrentTask:    status.CurrentTask,
			StartedAt:      status.StartedAt,
			UpdatedAt:      status.UpdatedAt,
		},
	})
}

func (h *SyncHandler) RepositoryList(c *gin.Context) {
	items, err := h.service.ListRepositories(c.Request.Context())
	if err != nil {
		klog.Errorf("[sync.RepositoryList] 获取仓库列表失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.RepositoryListResponse{
		Code: "OK",
		Data: items,
	})
}

func (h *SyncHandler) DocumentList(c *gin.Context) {
	var req struct {
		RepositoryID uint `form:"repository_id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := h.service.ListDocuments(c.Request.Context(), req.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.DocumentList] 获取文档列表失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.DocumentListResponse{
		Code: "OK",
		Data: items,
	})
}

func (h *SyncHandler) TargetList(c *gin.Context) {
	targets, err := h.service.ListSyncTargets(c.Request.Context())
	if err != nil {
		klog.Errorf("[sync.TargetList] 获取同步地址失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items := make([]syncdto.SyncTargetItem, 0, len(targets))
	for _, target := range targets {
		items = append(items, syncdto.SyncTargetItem{
			ID:        target.ID,
			URL:       target.URL,
			CreatedAt: target.CreatedAt,
			UpdatedAt: target.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, syncdto.SyncTargetListResponse{
		Code: "OK",
		Data: items,
	})
}

func (h *SyncHandler) TargetSave(c *gin.Context) {
	var req syncdto.SyncTargetSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target, err := h.service.SaveSyncTarget(c.Request.Context(), req.URL)
	if err != nil {
		klog.Errorf("[sync.TargetSave] 保存同步地址失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, syncdto.SyncTargetSaveResponse{
		Code: "OK",
		Data: syncdto.SyncTargetItem{
			ID:        target.ID,
			URL:       target.URL,
			CreatedAt: target.CreatedAt,
			UpdatedAt: target.UpdatedAt,
		},
	})
}

func (h *SyncHandler) TargetDelete(c *gin.Context) {
	var req syncdto.SyncTargetDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.DeleteSyncTarget(c.Request.Context(), req.ID); err != nil {
		klog.Errorf("[sync.TargetDelete] 删除同步地址失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, syncdto.SyncTargetDeleteResponse{
		Code: "OK",
		Data: syncdto.SyncTargetDeleteData{
			ID: req.ID,
		},
	})
}

func (h *SyncHandler) PullExport(c *gin.Context) {
	var req syncdto.PullExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.service.BuildPullExportData(c.Request.Context(), req.RepositoryID, req.DocumentIDs)
	if err != nil {
		klog.Errorf("[sync.PullExport] 生成拉取数据失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.PullExportResponse{
		Code: "OK",
		Data: data,
	})
}

// RepositoryUpsert 创建或更新仓库信息。
func (h *SyncHandler) RepositoryUpsert(c *gin.Context) {
	var req syncdto.RepositoryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo, err := h.service.CreateOrUpdateRepository(c.Request.Context(), req)
	if err != nil {
		klog.Errorf("[sync.RepositoryUpsert] 创建或更新仓库失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.RepositoryUpsertResponse{
		Code: "OK",
		Data: syncdto.RepositoryUpsertData{
			RepositoryID: repo.ID,
			Name:         repo.Name,
		},
	})
}

func (h *SyncHandler) RepositoryClear(c *gin.Context) {
	var req syncdto.RepositoryClearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ClearRepositoryData(c.Request.Context(), req.RepositoryID); err != nil {
		klog.Errorf("[sync.RepositoryClear] 清空仓库数据失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.RepositoryClearResponse{
		Code: "OK",
		Data: syncdto.RepositoryClearData{
			RepositoryID: req.RepositoryID,
		},
	})
}

func (h *SyncHandler) TaskCreate(c *gin.Context) {
	var req syncdto.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.CreateTask(c.Request.Context(), req)
	if err != nil {
		klog.Errorf("[sync.TaskCreate] 创建任务失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.TaskCreateResponse{
		Code: "OK",
		Data: syncdto.TaskCreateData{
			TaskID:       task.ID,
			RepositoryID: task.RepositoryID,
			Title:        task.Title,
		},
	})
}

func (h *SyncHandler) DocumentCreate(c *gin.Context) {
	var req syncdto.DocumentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.service.CreateDocument(c.Request.Context(), req)
	if err != nil {
		klog.Errorf("[sync.DocumentCreate] 创建文档失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.DocumentCreateResponse{
		Code: "OK",
		Data: syncdto.DocumentCreateData{
			DocumentID:   doc.ID,
			RepositoryID: doc.RepositoryID,
			TaskID:       doc.TaskID,
		},
	})
}

func (h *SyncHandler) TaskUpdateDocID(c *gin.Context) {
	var req syncdto.TaskUpdateDocIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.UpdateTaskDocID(c.Request.Context(), req.TaskID, req.DocumentID)
	if err != nil {
		klog.Errorf("[sync.TaskUpdateDocID] 更新任务文档ID失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.TaskUpdateDocIDResponse{
		Code: "OK",
		Data: syncdto.TaskUpdateDocIDData{
			TaskID:     task.ID,
			DocumentID: task.DocID,
		},
	})
}

// TaskUsageCreate 创建或覆盖任务用量记录
func (h *SyncHandler) TaskUsageCreate(c *gin.Context) {
	var req syncdto.TaskUsageCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usage, err := h.service.CreateTaskUsage(c.Request.Context(), req)
	if err != nil {
		klog.Errorf("[sync.TaskUsageCreate] 创建任务用量失败: error=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, syncdto.TaskUsageCreateResponse{
		Code: "OK",
		Data: syncdto.TaskUsageCreateData{
			TaskID: usage.TaskID,
		},
	})
}
