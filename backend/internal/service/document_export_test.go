package service

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockExportDocRepo struct {
	GetByRepositoryFunc func(repoID uint) ([]model.Document, error)
	GetAllLatestFunc    func() ([]model.Document, error)
}

func (m *mockExportDocRepo) Create(doc *model.Document) error {
	return nil
}
func (m *mockExportDocRepo) GetAllDocumentsTitleAndID(repoID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportDocRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	if m.GetByRepositoryFunc != nil {
		return m.GetByRepositoryFunc(repoID)
	}
	return nil, nil
}

func (m *mockExportDocRepo) GetVersions(repoID uint, title string) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportDocRepo) Get(id uint) (*model.Document, error) {
	return nil, nil
}

func (m *mockExportDocRepo) Save(doc *model.Document) error {
	return nil
}

func (m *mockExportDocRepo) Delete(id uint) error {
	return nil
}

func (m *mockExportDocRepo) DeleteByTaskID(taskID uint) error {
	return nil
}

func (m *mockExportDocRepo) DeleteByRepositoryID(repoID uint) error {
	return nil
}

func (m *mockExportDocRepo) UpdateTaskID(docID uint, taskID uint) error {
	return nil
}

func (m *mockExportDocRepo) TransferLatest(oldDocID uint, newDocID uint) error {
	return nil
}

func (m *mockExportDocRepo) CreateVersioned(doc *model.Document) error {
	return nil
}

func (m *mockExportDocRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	return 0, nil
}

func (m *mockExportDocRepo) ClearLatestByTaskID(taskID uint) error {
	return nil
}

func (m *mockExportDocRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportDocRepo) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	return nil, nil
}

func (m *mockExportDocRepo) GetAllLatest() ([]model.Document, error) {
	if m.GetAllLatestFunc != nil {
		return m.GetAllLatestFunc()
	}
	return nil, errors.New("GetAllLatest not implemented in mock")
}

type mockExportRepoRepo struct {
	GetBasicFunc func(id uint) (*model.Repository, error)
}

func (m *mockExportRepoRepo) Create(repo *model.Repository) error {
	return nil
}

func (m *mockExportRepoRepo) List() ([]model.Repository, error) {
	return nil, nil
}

func (m *mockExportRepoRepo) Get(id uint) (*model.Repository, error) {
	return nil, nil
}
func (m *mockExportRepoRepo) GetAllDocumentsTitleAndID(repoID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockExportRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.GetBasicFunc != nil {
		return m.GetBasicFunc(id)
	}
	return nil, errors.New("not found")
}

func (m *mockExportRepoRepo) Save(repo *model.Repository) error {
	return nil
}

func (m *mockExportRepoRepo) Delete(id uint) error {
	return nil
}

func TestDocumentServiceExportPDF(t *testing.T) {
	docRepo := &mockExportDocRepo{
		GetByRepositoryFunc: func(repoID uint) ([]model.Document, error) {
			return []model.Document{
				{Title: "概览", Content: "hello"},
				{Title: "架构", Content: ""},
			}, nil
		},
	}
	repoRepo := &mockExportRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return &model.Repository{ID: id, Name: "demo"}, nil
		},
	}
	service := NewDocumentService(&config.Config{}, docRepo, repoRepo, nil, eventbus.NewDocEventBus())

	data, filename, err := service.ExportPDF(1)
	if err != nil {
		t.Fatalf("ExportPDF error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected pdf data, got empty")
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("unexpected pdf header")
	}
	if !strings.HasSuffix(filename, ".pdf") {
		t.Fatalf("unexpected filename: %s", filename)
	}
}

func TestDocumentServiceExportPDFNoDocs(t *testing.T) {
	docRepo := &mockExportDocRepo{
		GetByRepositoryFunc: func(repoID uint) ([]model.Document, error) {
			return []model.Document{}, nil
		},
	}
	repoRepo := &mockExportRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return &model.Repository{ID: id, Name: "demo"}, nil
		},
	}
	service := NewDocumentService(&config.Config{}, docRepo, repoRepo, nil, eventbus.NewDocEventBus())

	_, _, err := service.ExportPDF(2)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
