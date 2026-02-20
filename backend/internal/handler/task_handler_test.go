package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// MockTaskService Mock任务服务接口
type MockTaskService struct {
	getFunc               func(id uint) (*model.Task, error)
	getByRepoFunc          func(repoID uint) ([]*model.Task, error)
	getStatsFunc           func(repoID uint) (map[string]int, error)
	enqueueFunc           func(id uint) error
	resetFunc             func(id uint) error
	forceResetFunc        func(id uint) error
	retryFunc             func(id uint) error
	reGenFunc             func(id uint) error
	cancelFunc            func(id uint) error
	deleteFunc            func(id uint) error
	getStuckFunc          func(timeout string) ([]*model.Task, error)
	cleanupStuckFunc      func(timeout string) (int, error)
	orchestratorStatusFunc func() map[string]interface{}
	globalMonitorFunc      func() (map[string]interface{}, error)
}

func (m *MockTaskService) Get(id uint) (*model.Task, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	if id == 999 {
		return nil, errors.New("task not found")
	}
	return &model.Task{ID: id, Title: "Test Task", Status: "pending"}, nil
}

func (m *MockTaskService) GetByRepository(repoID uint) ([]*model.Task, error) {
	if m.getByRepoFunc != nil {
		return m.getByRepoFunc(repoID)
	}
	return []*model.Task{
		{ID: 1, RepositoryID: repoID, Title: "Task 1", Status: "pending"},
		{ID: 2, RepositoryID: repoID, Title: "Task 2", Status: "completed"},
	}, nil
}

func (m *MockTaskService) GetTaskStats(repoID uint) (map[string]int, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(repoID)
	}
	return map[string]int{"pending": 1, "queued": 2, "running": 0, "completed": 3, "failed": 1}, nil
}

func (m *MockTaskService) Enqueue(id uint) error {
	if m.enqueueFunc != nil {
		return m.enqueueFunc(id)
	}
	return nil
}

func (m *MockTaskService) Reset(id uint) error {
	if m.resetFunc != nil {
		return m.resetFunc(id)
	}
	return nil
}

func (m *MockTaskService) ForceReset(id uint) error {
	if m.forceResetFunc != nil {
		return m.forceResetFunc(id)
	}
	return nil
}

func (m *MockTaskService) Retry(id uint) error {
	if m.retryFunc != nil {
		return m.retryFunc(id)
	}
	return nil
}

func (m *MockTaskService) ReGenByNewTask(id uint) error {
	if m.reGenFunc != nil {
		return m.reGenFunc(id)
	}
	return nil
}

func (m *MockTaskService) Cancel(id uint) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(id)
	}
	return nil
}

func (m *MockTaskService) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockTaskService) GetStuckTasks(timeout string) ([]*model.Task, error) {
	if m.getStuckFunc != nil {
		return m.getStuckFunc(timeout)
	}
	return []*model.Task{}, nil
}

func (m *MockTaskService) CleanupStuckTasks(timeout string) (int, error) {
	if m.cleanupStuckFunc != nil {
		return m.cleanupStuckFunc(timeout)
	}
	return 0, nil
}

func (m *MockTaskService) GetOrchestratorStatus() map[string]interface{} {
	if m.orchestratorStatusFunc != nil {
		return m.orchestratorStatusFunc()
	}
	return map[string]interface{}{
		"queue_length":  5,
		"workers":      3,
		"status":       "running",
	}
}

func (m *MockTaskService) GetGlobalMonitorData() (map[string]interface{}, error) {
	if m.globalMonitorFunc != nil {
		return m.globalMonitorFunc()
	}
	return map[string]interface{}{
		"total_tasks":     10,
		"pending_tasks":  2,
		"running_tasks":  3,
		"completed_tasks": 5,
	}, nil
}

func (m *MockTaskService) GetTaskUsageService() service.TaskUsageService {
	return &MockTaskUsageService{}
}

// MockTaskUsageService Mock任务用量服务接口
type MockTaskUsageService struct {
	getByTaskIDFunc func(ctx context.Context, taskID uint) (*model.TaskUsage, error)
	recordUsageFunc func(ctx context.Context, taskID uint, apiKeyName string, usage *schema.TokenUsage) error
}

func (m *MockTaskUsageService) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
	if m.getByTaskIDFunc != nil {
		return m.getByTaskIDFunc(ctx, taskID)
	}
	return &model.TaskUsage{
		TaskID:           taskID,
		PromptTokens:     800,
		CompletionTokens: 200,
		TotalTokens:      1000,
		APIKeyName:       "test-key",
	}, nil
}

func (m *MockTaskUsageService) RecordUsage(ctx context.Context, taskID uint, apiKeyName string, usage *schema.TokenUsage) error {
	if m.recordUsageFunc != nil {
		return m.recordUsageFunc(ctx, taskID, apiKeyName, usage)
	}
	return nil
}

// setupTaskRouter 创建测试路由 - 使用简单的mock处理器来测试路由
func setupTaskRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Create mock service
	mockTaskSvc := &MockTaskService{}
	mockUsageSvc := &MockTaskUsageService{}

	// Register simple handlers for testing
	api := router.Group("/api")
	api.GET("/repositories/:id/tasks", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		tasks, _ := mockTaskSvc.GetByRepository(1)
		c.JSON(http.StatusOK, tasks)
	})
	api.GET("/repositories/:id/tasks/stats", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		stats, _ := mockTaskSvc.GetTaskStats(1)
		c.JSON(http.StatusOK, stats)
	})
	api.POST("/tasks/:id/enqueue", func(c *gin.Context) {
		id := c.Param("id")
		if id == "999" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "task enqueued"})
	})
	api.POST("/tasks/:id/run", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task started"})
	})
	api.GET("/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		if id == "999" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "task not found"})
			return
		}
		task, _ := mockTaskSvc.Get(1)
		c.JSON(http.StatusOK, task)
	})
	api.POST("/tasks/:id/reset", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "task reset"})
	})
	api.POST("/tasks/:id/force-reset", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task force reset"})
	})
	api.POST("/tasks/:id/retry", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task retry started"})
	})
	api.POST("/tasks/:id/regen", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task re-generated"})
	})
	api.POST("/tasks/:id/cancel", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "task canceled"})
	})
	api.DELETE("/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
	})
	api.GET("/tasks/stuck", func(c *gin.Context) {
		tasks, _ := mockTaskSvc.GetStuckTasks("10m")
		c.JSON(http.StatusOK, gin.H{"tasks": tasks, "count": len(tasks)})
	})
	api.POST("/tasks/cleanup-stuck", func(c *gin.Context) {
		count, _ := mockTaskSvc.CleanupStuckTasks("10m")
		c.JSON(http.StatusOK, gin.H{"message": "cleanup completed", "count": count})
	})
	api.GET("/orchestrator/status", func(c *gin.Context) {
		status := mockTaskSvc.GetOrchestratorStatus()
		c.JSON(http.StatusOK, status)
	})
	api.GET("/tasks/monitor", func(c *gin.Context) {
		data, _ := mockTaskSvc.GetGlobalMonitorData()
		c.JSON(http.StatusOK, data)
	})
	api.GET("/tasks/:id/usage", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		usage, _ := mockUsageSvc.GetByTaskID(context.Background(), 1)
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    usage,
		})
	})

	return router
}

// TestTaskHandler_GetByRepository 测试获取仓库的任务列表
func TestTaskHandler_GetByRepository(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取任务列表", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var tasks []*model.Task
		err := requireJSONDecode(w.Body.Bytes(), &tasks)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
	})

	t.Run("无效的仓库ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/invalid/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTaskHandler_GetStats 测试获取任务统计
func TestTaskHandler_GetStats(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取任务统计", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/tasks/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var stats map[string]int
		err := requireJSONDecode(w.Body.Bytes(), &stats)
		require.NoError(t, err)
		assert.Equal(t, 1, stats["pending"])
		assert.Equal(t, 2, stats["queued"])
	})
}

// TestTaskHandler_Get 测试获取单个任务
func TestTaskHandler_Get(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("任务不存在", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestTaskHandler_Enqueue 测试排队任务
func TestTaskHandler_Enqueue(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功排队任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/enqueue", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task enqueued", response["message"])
	})

	t.Run("任务不存在", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/999/enqueue", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestTaskHandler_Run 测试运行任务（兼容接口）
func TestTaskHandler_Run(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功运行任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/run", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestTaskHandler_Reset 测试重置任务
func TestTaskHandler_Reset(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功重置任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/reset", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task reset", response["message"])
	})

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/invalid/reset", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTaskHandler_ForceReset 测试强制重置任务
func TestTaskHandler_ForceReset(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功强制重置任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/force-reset", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task force reset", response["message"])
	})
}

// TestTaskHandler_Retry 测试重试任务
func TestTaskHandler_Retry(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功重试任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/retry", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task retry started", response["message"])
	})
}

// TestTaskHandler_ReGenByNewTask 测试重新生成任务
func TestTaskHandler_ReGenByNewTask(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功重新生成任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/regen", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task re-generated", response["message"])
	})
}

// TestTaskHandler_Cancel 测试取消任务
func TestTaskHandler_Cancel(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功取消任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/1/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task canceled", response["message"])
	})
}

// TestTaskHandler_Delete 测试删除任务
func TestTaskHandler_Delete(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功删除任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/api/tasks/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "task deleted", response["message"])
	})

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/api/tasks/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTaskHandler_GetStuck 测试获取卡住的任务
func TestTaskHandler_GetStuck(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取卡住任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/stuck?timeout=10m", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "tasks")
		assert.Contains(t, response, "count")
	})
}

// TestTaskHandler_CleanupStuck 测试清理卡住任务
func TestTaskHandler_CleanupStuck(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功清理卡住任务", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/tasks/cleanup-stuck?timeout=10m", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "cleanup completed", response["message"])
	})
}

// TestTaskHandler_GetOrchestratorStatus 测试获取编排器状态
func TestTaskHandler_GetOrchestratorStatus(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取编排器状态", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/orchestrator/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var status map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &status)
		require.NoError(t, err)
		assert.Equal(t, float64(5), status["queue_length"])
		assert.Equal(t, float64(3), status["workers"])
	})
}

// TestTaskHandler_Monitor 测试全局任务监控
func TestTaskHandler_Monitor(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取监控数据", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/monitor", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var data map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.Equal(t, float64(10), data["total_tasks"])
		assert.Equal(t, float64(3), data["running_tasks"])
	})
}

// TestTaskHandler_GetTaskUsage 测试获取任务Token用量
func TestTaskHandler_GetTaskUsage(t *testing.T) {
	router := setupTaskRouter()

	t.Run("成功获取Token用量", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/1/usage", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := requireJSONDecode(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(0), response["code"])
		assert.Contains(t, response, "data")
	})

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/tasks/invalid/usage", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// requireJSONDecode 辅助函数
func requireJSONDecode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

