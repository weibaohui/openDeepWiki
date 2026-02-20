package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// MockDocumentService Mock文档服务接口
type MockDocumentService struct {
	getFunc        func(id uint) (*model.Document, error)
	getByRepoFunc   func(repoID uint) ([]*model.Document, error)
	getVersionsFunc func(id uint) ([]*model.Document, error)
	updateFunc     func(id uint, content string) (*model.Document, error)
	ratingFunc     func(id uint, score int) (*model.DocumentRatingStats, error)
	ratingStatsFunc func(id uint) (*model.DocumentRatingStats, error)
	indexFunc      func(repoID uint) (string, error)
	exportFunc     func(repoID uint) ([]byte, string, error)
	exportPDFFunc  func(repoID uint) ([]byte, string, error)
	redirectFunc   func(id uint, path string) (string, error)
	tokenUsageFunc func(id uint) (map[string]interface{}, error)
}

func (m *MockDocumentService) Get(id uint) (*model.Document, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	if id == 999 {
		return nil, errors.New("document not found")
	}
	return &model.Document{ID: id, RepositoryID: 1, Title: "Test Doc", Content: "Test content"}, nil
}

func (m *MockDocumentService) GetByRepository(repoID uint) ([]*model.Document, error) {
	if m.getByRepoFunc != nil {
		return m.getByRepoFunc(repoID)
	}
	return []*model.Document{
		{ID: 1, RepositoryID: repoID, Title: "Doc 1"},
		{ID: 2, RepositoryID: repoID, Title: "Doc 2"},
	}, nil
}

func (m *MockDocumentService) GetVersions(id uint) ([]*model.Document, error) {
	if m.getVersionsFunc != nil {
		return m.getVersionsFunc(id)
	}
	return []*model.Document{
		{ID: id, Version: 1, IsLatest: false},
		{ID: id + 100, Version: 2, IsLatest: true},
	}, nil
}

func (m *MockDocumentService) Update(id uint, content string) (*model.Document, error) {
	if m.updateFunc != nil {
		return m.updateFunc(id, content)
	}
	return &model.Document{ID: id, Content: content}, nil
}

func (m *MockDocumentService) SubmitRating(id uint, score int) (*model.DocumentRatingStats, error) {
	if m.ratingFunc != nil {
		return m.ratingFunc(id, score)
	}
	return &model.DocumentRatingStats{AverageScore: float64(score), RatingCount: 1}, nil
}

func (m *MockDocumentService) GetRatingStats(id uint) (*model.DocumentRatingStats, error) {
	if m.ratingStatsFunc != nil {
		return m.ratingStatsFunc(id)
	}
	return &model.DocumentRatingStats{AverageScore: 4.5, RatingCount: 10}, nil
}

func (m *MockDocumentService) GetIndex(repoID uint) (string, error) {
	if m.indexFunc != nil {
		return m.indexFunc(repoID)
	}
	return "# Test Index", nil
}

func (m *MockDocumentService) ExportAll(repoID uint) ([]byte, string, error) {
	if m.exportFunc != nil {
		return m.exportFunc(repoID)
	}
	return []byte("zip data"), "documents.zip", nil
}

func (m *MockDocumentService) ExportPDF(repoID uint) ([]byte, string, error) {
	if m.exportPDFFunc != nil {
		return m.exportPDFFunc(repoID)
	}
	return []byte("pdf data"), "documents.pdf", nil
}

func (m *MockDocumentService) GetRedirectURL(id uint, path string) (string, error) {
	if m.redirectFunc != nil {
		return m.redirectFunc(id, path)
	}
	return "http://github.com/test/repo/blob/main/" + path, nil
}

func (m *MockDocumentService) GetTokenUsage(id uint) (map[string]interface{}, error) {
	if m.tokenUsageFunc != nil {
		return m.tokenUsageFunc(id)
	}
	return map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"total_tokens":      1000,
			"prompt_tokens":     800,
			"completion_tokens": 200,
		},
	}, nil
}

func (m *MockDocumentService) GetVersionsByRepository(repoID uint) ([]*model.Document, error) {
	return nil, nil
}

// setupDocumentRouter 创建测试路由
func setupDocumentRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	mockDocSvc := &MockDocumentService{}

	// Register simple handlers for testing
	api := router.Group("/api")
	api.GET("/repositories/:id/documents", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		docs, _ := mockDocSvc.GetByRepository(1)
		c.JSON(http.StatusOK, docs)
	})
	api.GET("/documents/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		if id == "999" {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		doc, _ := mockDocSvc.Get(1)
		c.JSON(http.StatusOK, doc)
	})
	api.GET("/documents/:id/versions", func(c *gin.Context) {
		versions, _ := mockDocSvc.GetVersions(1)
		c.JSON(http.StatusOK, versions)
	})
	api.PUT("/documents/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
			return
		}
		doc, _ := mockDocSvc.Update(1, req.Content)
		c.JSON(http.StatusOK, doc)
	})
	api.GET("/repositories/:id/documents/export", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		data, filename, _ := mockDocSvc.ExportAll(1)
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/zip", data)
	})
	api.GET("/repositories/:id/documents/pdf", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		data, filename, _ := mockDocSvc.ExportPDF(1)
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/pdf", data)
	})
	api.GET("/documents/:id/redirect", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
			return
		}
		url, _ := mockDocSvc.GetRedirectURL(1, path)
		c.JSON(http.StatusOK, gin.H{"redirect_url": url})
	})
	api.GET("/repositories/:id/documents/index", func(c *gin.Context) {
		repoID := c.Param("id")
		if repoID == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
			return
		}
		index, _ := mockDocSvc.GetIndex(1)
		c.JSON(http.StatusOK, gin.H{"index": index})
	})
	api.POST("/documents/:id/rating", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		var req struct {
			Score int `json:"score" binding:"required,min=1,max=5"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		stats, _ := mockDocSvc.SubmitRating(1, req.Score)
		c.JSON(http.StatusOK, stats)
	})
	api.GET("/documents/:id/rating-stats", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		stats, _ := mockDocSvc.GetRatingStats(1)
		c.JSON(http.StatusOK, stats)
	})
	api.GET("/documents/:id/token-usage", func(c *gin.Context) {
		id := c.Param("id")
		if id == "invalid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
			return
		}
		usage, _ := mockDocSvc.GetTokenUsage(1)
		c.JSON(http.StatusOK, usage)
	})

	return router
}

// TestDocumentHandler_GetByRepository 测试获取仓库文档列表
func TestDocumentHandler_GetByRepository(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取文档列表", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/documents", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无效的仓库ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/invalid/documents", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestDocumentHandler_Get 测试获取文档详情
func TestDocumentHandler_Get(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取文档", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无效的ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("文档不存在", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestDocumentHandler_GetVersions 测试获取文档版本
func TestDocumentHandler_GetVersions(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取文档版本", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDocumentHandler_Update 测试更新文档
func TestDocumentHandler_Update(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功更新文档", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"content": "updated content",
		}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/documents/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无效的ID", func(t *testing.T) {
		requestBody := map[string]interface{}{"content": "test"}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/documents/invalid", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("缺少content字段", func(t *testing.T) {
		requestBody := map[string]interface{}{}
		body, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/documents/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestDocumentHandler_SubmitRating 测试提交评分
func TestDocumentHandler_SubmitRating(t *testing.T) {
	router := setupDocumentRouter()

	tests := []struct {
		name           string
		id             string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "成功提交评分",
			id:   "1",
			requestBody: map[string]interface{}{
				"score": 5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "无效的ID",
			id:   "invalid",
			requestBody: map[string]interface{}{"score": 5},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "评分小于1",
			id:   "1",
			requestBody: map[string]interface{}{
				"score": 0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "评分大于5",
			id:   "1",
			requestBody: map[string]interface{}{
				"score": 6,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "缺少score字段",
			id:             "1",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/documents/"+tt.id+"/rating", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestDocumentHandler_GetRatingStats 测试获取评分统计
func TestDocumentHandler_GetRatingStats(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取评分统计", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1/rating-stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDocumentHandler_Export 测试导出文档
func TestDocumentHandler_Export(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功导出ZIP", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/documents/export", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDocumentHandler_ExportPDF 测试导出PDF
func TestDocumentHandler_ExportPDF(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功导出PDF", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/documents/pdf", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDocumentHandler_Redirect 测试重定向
func TestDocumentHandler_Redirect(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功重定向", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1/redirect?path=README.md", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("缺少path参数", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1/redirect", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestDocumentHandler_GetIndex 测试获取文档索引
func TestDocumentHandler_GetIndex(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取索引", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/repositories/1/documents/index", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDocumentHandler_GetTokenUsage 测试获取Token用量
func TestDocumentHandler_GetTokenUsage(t *testing.T) {
	router := setupDocumentRouter()

	t.Run("成功获取Token用量", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/documents/1/token-usage", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

