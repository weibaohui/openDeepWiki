package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type mockRatingRepo struct {
	CreateFunc   func(rating *model.DocumentRating) error
	GetLatest    func(documentID uint) (*model.DocumentRating, error)
	GetStatsFunc func(documentID uint) (*model.DocumentRatingStats, error)
}

// Create 创建评分记录
func (m *mockRatingRepo) Create(rating *model.DocumentRating) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(rating)
	}
	return nil
}

// GetLatestByDocumentID 获取最新评分
func (m *mockRatingRepo) GetLatestByDocumentID(documentID uint) (*model.DocumentRating, error) {
	if m.GetLatest != nil {
		return m.GetLatest(documentID)
	}
	return nil, nil
}

// GetStatsByDocumentID 获取评分统计
func (m *mockRatingRepo) GetStatsByDocumentID(documentID uint) (*model.DocumentRatingStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(documentID)
	}
	return &model.DocumentRatingStats{}, nil
}

// TestDocumentHandlerSubmitRating 验证评分提交接口返回统计结果
func TestDocumentHandlerSubmitRating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ratingRepo := &mockRatingRepo{
		GetStatsFunc: func(documentID uint) (*model.DocumentRatingStats, error) {
			return &model.DocumentRatingStats{
				AverageScore: 4.5,
				RatingCount:  10,
			}, nil
		},
	}
	docService := service.NewDocumentService(&config.Config{}, nil, nil, ratingRepo)
	handler := NewDocumentHandler(nil, docService)
	router := gin.New()
	router.POST("/documents/:id/ratings", handler.SubmitRating)

	body := []byte(`{"score":4}`)
	req := httptest.NewRequest(http.MethodPost, "/documents/12/ratings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var payload model.DocumentRatingStats
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if payload.RatingCount != 10 {
		t.Fatalf("unexpected rating count: %d", payload.RatingCount)
	}
}

// TestDocumentHandlerGetRatingStats 验证评分统计接口返回统计结果
func TestDocumentHandlerGetRatingStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ratingRepo := &mockRatingRepo{
		GetStatsFunc: func(documentID uint) (*model.DocumentRatingStats, error) {
			return &model.DocumentRatingStats{
				AverageScore: 3.6,
				RatingCount:  6,
			}, nil
		},
	}
	docService := service.NewDocumentService(&config.Config{}, nil, nil, ratingRepo)
	handler := NewDocumentHandler(nil, docService)
	router := gin.New()
	router.GET("/documents/:id/ratings/stats", handler.GetRatingStats)

	req := httptest.NewRequest(http.MethodGet, "/documents/8/ratings/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var payload model.DocumentRatingStats
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if payload.AverageScore != 3.6 || payload.RatingCount != 6 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
