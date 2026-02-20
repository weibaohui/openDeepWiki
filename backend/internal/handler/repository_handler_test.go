package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// TestRepositoryHandler_Create 测试创建仓库
func TestRepositoryHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	// Mock event buses
	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	// We can't easily mock the RepositoryService, so we'll just test the route registration
	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)

	router.POST("/api/v1/repositories", h.Create)

	t.Run("缺少url字段", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name": "test-repo",
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "error")
	})
}

// TestRepositoryHandler_List 测试列出仓库
func TestRepositoryHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.GET("/api/v1/repositories", h.List)

	t.Run("服务为nil时的响应", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Without a service, this will fail
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestRepositoryHandler_Get 测试获取仓库详情
func TestRepositoryHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.GET("/api/v1/repositories/:id", h.Get)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
		verifyResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "无效的ID",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid id")
			},
		},
		{
			name:           "ID为0",
			id:             "0",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "仓库不存在（服务为nil）",
			id:             "999",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/repositories/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.verifyResponse != nil {
				tt.verifyResponse(t, w)
			}
		})
	}
}

// TestRepositoryHandler_Delete 测试删除仓库
func TestRepositoryHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.DELETE("/api/v1/repositories/:id", h.Delete)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{
			name:           "无效的ID",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "删除失败（服务为nil）",
			id:             "1",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/repositories/"+tt.id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestRepositoryHandler_RunAllTasks 测试运行所有任务
func TestRepositoryHandler_RunAllTasks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/run-all", h.RunAllTasks)

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/invalid/run-all", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("服务为nil时", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/run-all", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestRepositoryHandler_AnalyzeDirectory 测试分析目录
func TestRepositoryHandler_AnalyzeDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/analyze-directory", h.AnalyzeDirectory)

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/invalid/analyze-directory", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("成功触发分析", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/analyze-directory", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "message")
	})
}

// TestRepositoryHandler_AnalyzeAPI 测试分析API
func TestRepositoryHandler_AnalyzeAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/analyze-api", h.AnalyzeAPI)

	t.Run("成功触发API分析", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/analyze-api", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "message")
	})
}

// TestRepositoryHandler_AnalyzeDatabaseModel 测试分析数据库模型
func TestRepositoryHandler_AnalyzeDatabaseModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/analyze-database-model", h.AnalyzeDatabaseModel)

	t.Run("成功触发数据库模型分析", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/analyze-database-model", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestRepositoryHandler_IncrementalAnalysis 测试增量分析
func TestRepositoryHandler_IncrementalAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/incremental-analysis", h.IncrementalAnalysis)

	t.Run("成功触发增量分析", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/incremental-analysis", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "message")
	})
}

// TestRepositoryHandler_SetReady 测试设置就绪状态
func TestRepositoryHandler_SetReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/set-ready", h.SetReady)

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/invalid/set-ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("服务为nil时", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/set-ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestRepositoryHandler_Clone 测试重新克隆仓库
func TestRepositoryHandler_Clone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/clone", h.Clone)

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/invalid/clone", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("服务为nil时", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/clone", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestRepositoryHandler_PurgeLocal 测试清空本地目录
func TestRepositoryHandler_PurgeLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.POST("/api/v1/repositories/:id/purge-local", h.PurgeLocal)

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/invalid/purge-local", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("服务为nil时", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/repositories/1/purge-local", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// TestRepositoryHandler_GetIncrementalHistory 测试获取增量历史
func TestRepositoryHandler_GetIncrementalHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add recovery middleware to handle panics from nil service
	router.Use(gin.Recovery())

	repoBus := eventbus.NewRepositoryEventBus()
	taskBus := eventbus.NewTaskEventBus()

	h := NewRepositoryHandler(repoBus, taskBus, nil, nil)
	router.GET("/api/v1/repositories/:id/incremental-history", h.GetIncrementalHistory)

	tests := []struct {
		name           string
		id             string
		query          string
		expectedStatus int
	}{
		{
			name:           "无效的ID",
			id:             "invalid",
			query:          "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "服务为nil时",
			id:             "1",
			query:          "?limit=10",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "服务为nil时（默认limit）",
			id:             "1",
			query:          "",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/repositories/"+tt.id+"/incremental-history"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestRepositoryHandler_AnalyzeProblemRequest 测试分析问题请求结构
func TestRepositoryHandler_AnalyzeProblemRequest(t *testing.T) {
	// Just verify the request struct can be marshaled/unmarshaled correctly
	req := AnalyzeProblemRequest{
		Content: "test content",
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded AnalyzeProblemRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "test content", decoded.Content)
}

// TestServiceErrors 测试服务错误类型
func TestServiceErrors(t *testing.T) {
	t.Run("ErrInvalidRepositoryURL", func(t *testing.T) {
		assert.Equal(t, "invalid repository url", service.ErrInvalidRepositoryURL.Error())
	})

	t.Run("ErrRepositoryAlreadyExists", func(t *testing.T) {
		assert.Equal(t, "repository already exists", service.ErrRepositoryAlreadyExists.Error())
	})
}
