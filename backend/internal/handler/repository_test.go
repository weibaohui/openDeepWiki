package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type mockRepoRepo struct {
	CreateFunc   func(repo *model.Repository) error
	ListFunc     func() ([]model.Repository, error)
	GetBasicFunc func(id uint) (*model.Repository, error)
	SaveFunc     func(repo *model.Repository) error
}

func (m *mockRepoRepo) Create(repo *model.Repository) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(repo)
	}
	return nil
}

func (m *mockRepoRepo) List() ([]model.Repository, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil, nil
}

func (m *mockRepoRepo) Get(id uint) (*model.Repository, error) {
	return nil, nil
}

func (m *mockRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.GetBasicFunc != nil {
		return m.GetBasicFunc(id)
	}
	return nil, nil
}

func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(repo)
	}
	return nil
}

func (m *mockRepoRepo) Delete(id uint) error {
	return nil
}

type mockTaskRepo struct{}

func (m *mockTaskRepo) Create(task *model.Task) error {
	return nil
}

func (m *mockTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) Get(id uint) (*model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) Save(task *model.Task) error {
	return nil
}

func (m *mockTaskRepo) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockTaskRepo) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) DeleteByRepositoryID(repoID uint) error {
	return nil
}

func (m *mockTaskRepo) Delete(id uint) error {
	return nil
}

func (m *mockTaskRepo) GetTaskStats(repoID uint) (map[string]int64, error) {
	return nil, nil
}

type mockDocumentRepo struct{}

func (m *mockDocumentRepo) Create(doc *model.Document) error {
	return nil
}

func (m *mockDocumentRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockDocumentRepo) Get(id uint) (*model.Document, error) {
	return nil, nil
}

func (m *mockDocumentRepo) Save(doc *model.Document) error {
	return nil
}

func (m *mockDocumentRepo) Delete(id uint) error {
	return nil
}

func (m *mockDocumentRepo) DeleteByTaskID(taskID uint) error {
	return nil
}

func (m *mockDocumentRepo) DeleteByRepositoryID(repoID uint) error {
	return nil
}

func (m *mockDocumentRepo) CreateVersioned(doc *model.Document) error {
	return nil
}

func (m *mockDocumentRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	return 0, nil
}

func (m *mockDocumentRepo) ClearLatestByTaskID(taskID uint) error {
	return nil
}

func (m *mockDocumentRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	return nil, nil
}

type mockDirMakerService struct {
	CreateDirsFunc func(ctx context.Context, repo *model.Repository) ([]*model.Task, error)
}

func (m *mockDirMakerService) CreateDirs(ctx context.Context, repo *model.Repository) ([]*model.Task, error) {
	if m.CreateDirsFunc != nil {
		return m.CreateDirsFunc(ctx, repo)
	}
	return nil, nil
}

func TestRepositoryHandlerCreateInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repoRepo := &mockRepoRepo{}
	svc := service.NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
	handler := NewRepositoryHandler(svc)
	router := gin.New()
	router.POST("/repositories", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/repositories", strings.NewReader(`{"url":"https://github.com/owner/repo/blob/main/README.md"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRepositoryHandlerCreateDuplicateURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repoRepo := &mockRepoRepo{
		ListFunc: func() ([]model.Repository, error) {
			return []model.Repository{{ID: 1, URL: "https://github.com/Owner/Repo?tab=readme"}}, nil
		},
	}
	svc := service.NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
	handler := NewRepositoryHandler(svc)
	router := gin.New()
	router.POST("/repositories", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/repositories", strings.NewReader(`{"url":"https://github.com/owner/repo"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestRepositoryHandlerPurgeLocalSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir, err := os.MkdirTemp("", "purge-local-handler")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        1,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: tempDir,
	}

	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		SaveFunc: func(updated *model.Repository) error {
			repo = updated
			return nil
		},
	}

	svc := service.NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
	handler := NewRepositoryHandler(svc)
	router := gin.New()
	router.POST("/repositories/:id/purge-local", handler.PurgeLocal)

	req := httptest.NewRequest(http.MethodPost, "/repositories/1/purge-local", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("expected local dir removed, stat err=%v", err)
	}
}

func TestRepositoryHandlerPurgeLocalInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewRepositoryService(&config.Config{}, &mockRepoRepo{}, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
	handler := NewRepositoryHandler(svc)
	router := gin.New()
	router.POST("/repositories/:id/purge-local", handler.PurgeLocal)

	req := httptest.NewRequest(http.MethodPost, "/repositories/abc/purge-local", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRepositoryHandlerAnalyzeDirectoryStarted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &model.Repository{
		ID:     4,
		Status: string(statemachine.RepoStatusReady),
	}
	called := make(chan struct{}, 1)

	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	dirMaker := &mockDirMakerService{
		CreateDirsFunc: func(ctx context.Context, target *model.Repository) ([]*model.Task, error) {
			called <- struct{}{}
			return []*model.Task{{ID: 1}}, nil
		},
	}

	svc := service.NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, dirMaker)
	handler := NewRepositoryHandler(svc)
	router := gin.New()
	router.POST("/repositories/:id/directory-analyze", handler.AnalyzeDirectory)

	req := httptest.NewRequest(http.MethodPost, "/repositories/4/directory-analyze", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "directory analysis started") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}

	select {
	case <-called:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async directory analysis to be triggered")
	}
}
