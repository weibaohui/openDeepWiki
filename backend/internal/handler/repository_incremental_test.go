package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type mockRepoRepo struct {
	repos map[uint]*model.Repository
	err   error
}

func (m *mockRepoRepo) Create(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	if repo.ID == 0 {
		repo.ID = uint(len(m.repos) + 1)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockRepoRepo) List() ([]model.Repository, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Repository
	for _, repo := range m.repos {
		out = append(out, *repo)
	}
	return out, nil
}

func (m *mockRepoRepo) Get(id uint) (*model.Repository, error) {
	return m.GetBasic(id)
}

func (m *mockRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.err != nil {
		return nil, m.err
	}
	repo, ok := m.repos[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	return repo, nil
}

func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockRepoRepo) Delete(id uint) error {
	delete(m.repos, id)
	return nil
}

func TestRepositoryHandlerIncrementalAnalysisRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repoRepo := &mockRepoRepo{err: errors.New("db error")}
	svc := service.NewRepositoryService(&config.Config{}, repoRepo, nil, nil, nil)
	handler := NewRepositoryHandler(nil, nil, svc, nil)
	router := gin.New()
	router.POST("/repositories/:id/incremental-analysis", handler.IncrementalAnalysis)

	req := httptest.NewRequest(http.MethodPost, "/repositories/1/incremental-analysis", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
