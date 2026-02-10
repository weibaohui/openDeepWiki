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
		syncGroup.GET("/status/:sync_id", h.Status)
		syncGroup.POST("/repository-upsert", h.RepositoryUpsert)
		syncGroup.POST("/task-create", h.TaskCreate)
		syncGroup.POST("/document-create", h.DocumentCreate)
		syncGroup.POST("/task-update-docid", h.TaskUpdateDocID)
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

	status, err := h.service.Start(c.Request.Context(), req.TargetServer, req.RepositoryID, req.DocumentIDs)
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
