package service

import (
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockRatingRepo struct {
	CreateFunc     func(rating *model.DocumentRating) error
	GetLatestFunc  func(documentID uint) (*model.DocumentRating, error)
	GetStatsFunc   func(documentID uint) (*model.DocumentRatingStats, error)
	CreateCalled   int
	LatestCalled   int
	GetStatsCalled int
}

// Create 记录评分数据
func (m *mockRatingRepo) Create(rating *model.DocumentRating) error {
	m.CreateCalled++
	if m.CreateFunc != nil {
		return m.CreateFunc(rating)
	}
	return nil
}

// GetLatestByDocumentID 获取最新评分
func (m *mockRatingRepo) GetLatestByDocumentID(documentID uint) (*model.DocumentRating, error) {
	m.LatestCalled++
	if m.GetLatestFunc != nil {
		return m.GetLatestFunc(documentID)
	}
	return nil, nil
}

// GetStatsByDocumentID 获取评分统计
func (m *mockRatingRepo) GetStatsByDocumentID(documentID uint) (*model.DocumentRatingStats, error) {
	m.GetStatsCalled++
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(documentID)
	}
	return &model.DocumentRatingStats{}, nil
}

// TestDocumentServiceSubmitRatingCreate 验证评分提交成功
func TestDocumentServiceSubmitRatingCreate(t *testing.T) {
	ratingRepo := &mockRatingRepo{
		GetStatsFunc: func(documentID uint) (*model.DocumentRatingStats, error) {
			return &model.DocumentRatingStats{
				AverageScore: 4.25,
				RatingCount:  4,
			}, nil
		},
	}
	service := NewDocumentService(&config.Config{}, nil, nil, ratingRepo)

	stats, err := service.SubmitRating(10, 5)
	if err != nil {
		t.Fatalf("SubmitRating error: %v", err)
	}
	if ratingRepo.CreateCalled != 1 {
		t.Fatalf("expected Create to be called once, got %d", ratingRepo.CreateCalled)
	}
	if stats.AverageScore != 4.3 || stats.RatingCount != 4 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

// TestDocumentServiceSubmitRatingDuplicate 验证重复提交的评分被忽略
func TestDocumentServiceSubmitRatingDuplicate(t *testing.T) {
	ratingRepo := &mockRatingRepo{
		GetLatestFunc: func(documentID uint) (*model.DocumentRating, error) {
			return &model.DocumentRating{
				ID:         1,
				DocumentID: documentID,
				Score:      3,
				CreatedAt:  time.Now().Add(-5 * time.Second),
			}, nil
		},
		GetStatsFunc: func(documentID uint) (*model.DocumentRatingStats, error) {
			return &model.DocumentRatingStats{
				AverageScore: 3.5,
				RatingCount:  2,
			}, nil
		},
	}
	service := NewDocumentService(&config.Config{}, nil, nil, ratingRepo)

	stats, err := service.SubmitRating(22, 3)
	if err != nil {
		t.Fatalf("SubmitRating error: %v", err)
	}
	if ratingRepo.CreateCalled != 0 {
		t.Fatalf("expected Create not to be called, got %d", ratingRepo.CreateCalled)
	}
	if stats.RatingCount != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
