package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

type mockExportHandlerDocRepo struct {
	GetByRepositoryFunc func(repoID uint) ([]model.Document, error)
}

func (m *mockExportHandlerDocRepo) Create(doc *model.Document) error {
	return nil
}

func (m *mockExportHandlerDocRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	if m.GetByRepositoryFunc != nil {
		return m.GetByRepositoryFunc(repoID)
	}
	return nil, nil
}

func (m *mockExportHandlerDocRepo) GetVersions(repoID uint, title string) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportHandlerDocRepo) Get(id uint) (*model.Document, error) {
	return nil, nil
}

func (m *mockExportHandlerDocRepo) Save(doc *model.Document) error {
	return nil
}

func (m *mockExportHandlerDocRepo) Delete(id uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) DeleteByTaskID(taskID uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) DeleteByRepositoryID(repoID uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) UpdateTaskID(docID uint, taskID uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) TransferLatest(oldDocID uint, newDocID uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) CreateVersioned(doc *model.Document) error {
	return nil
}

func (m *mockExportHandlerDocRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	return 0, nil
}

func (m *mockExportHandlerDocRepo) ClearLatestByTaskID(taskID uint) error {
	return nil
}

func (m *mockExportHandlerDocRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportHandlerDocRepo) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	return nil, nil
}

type mockExportHandlerRepoRepo struct {
	GetBasicFunc func(id uint) (*model.Repository, error)
}

func (m *mockExportHandlerRepoRepo) Create(repo *model.Repository) error {
	return nil
}

func (m *mockExportHandlerRepoRepo) List() ([]model.Repository, error) {
	return nil, nil
}

func (m *mockExportHandlerRepoRepo) Get(id uint) (*model.Repository, error) {
	return nil, nil
}

func (m *mockExportHandlerRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.GetBasicFunc != nil {
		return m.GetBasicFunc(id)
	}
	return nil, nil
}

func (m *mockExportHandlerRepoRepo) Save(repo *model.Repository) error {
	return nil
}

func (m *mockExportHandlerRepoRepo) Delete(id uint) error {
	return nil
}

func TestDocumentHandlerExportPDF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	docRepo := &mockExportHandlerDocRepo{
		GetByRepositoryFunc: func(repoID uint) ([]model.Document, error) {
			return []model.Document{{Title: "概览", Content: "hello"}}, nil
		},
	}
	repoRepo := &mockExportHandlerRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return &model.Repository{ID: id, Name: "demo"}, nil
		},
	}
	docService := service.NewDocumentService(&config.Config{}, docRepo, repoRepo, nil)
	handler := NewDocumentHandler(nil, docService)
	router := gin.New()
	router.GET("/repositories/:id/export-pdf", handler.ExportPDF)

	req := httptest.NewRequest(http.MethodGet, "/repositories/1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/pdf") {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
	if w.Body.Len() == 0 {
		t.Fatalf("expected pdf data, got empty")
	}
}

func TestDocumentHandlerExportPDFInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	docService := service.NewDocumentService(&config.Config{}, &mockExportHandlerDocRepo{}, &mockExportHandlerRepoRepo{}, nil)
	handler := NewDocumentHandler(nil, docService)
	router := gin.New()
	router.GET("/repositories/:id/export-pdf", handler.ExportPDF)

	req := httptest.NewRequest(http.MethodGet, "/repositories/invalid/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
