package service

import (
	"errors"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockDocumentRepo struct {
	CreateFunc                 func(doc *model.Document) error
	GetByRepositoryFunc        func(repoID uint) ([]model.Document, error)
	GetFunc                    func(id uint) (*model.Document, error)
	SaveFunc                   func(doc *model.Document) error
	DeleteFunc                 func(id uint) error
	DeleteByTaskIDFunc         func(taskID uint) error
	DeleteByRepositoryIDFunc   func(repoID uint) error
	CreateVersionedFunc        func(doc *model.Document) error
	GetLatestVersionByTaskFunc func(taskID uint) (int, error)
	ClearLatestByTaskIDFunc    func(taskID uint) error
	GetByTaskIDFunc            func(taskID uint) ([]model.Document, error)
}

func (m *mockDocumentRepo) Create(doc *model.Document) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(doc)
	}
	return nil
}

func (m *mockDocumentRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	if m.GetByRepositoryFunc != nil {
		return m.GetByRepositoryFunc(repoID)
	}
	return nil, nil
}

func (m *mockDocumentRepo) Get(id uint) (*model.Document, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *mockDocumentRepo) Save(doc *model.Document) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(doc)
	}
	return nil
}

func (m *mockDocumentRepo) Delete(id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *mockDocumentRepo) DeleteByTaskID(taskID uint) error {
	if m.DeleteByTaskIDFunc != nil {
		return m.DeleteByTaskIDFunc(taskID)
	}
	return nil
}

func (m *mockDocumentRepo) DeleteByRepositoryID(repoID uint) error {
	if m.DeleteByRepositoryIDFunc != nil {
		return m.DeleteByRepositoryIDFunc(repoID)
	}
	return nil
}

func (m *mockDocumentRepo) CreateVersioned(doc *model.Document) error {
	if m.CreateVersionedFunc != nil {
		return m.CreateVersionedFunc(doc)
	}
	return nil
}

func (m *mockDocumentRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	if m.GetLatestVersionByTaskFunc != nil {
		return m.GetLatestVersionByTaskFunc(taskID)
	}
	return 0, nil
}

func (m *mockDocumentRepo) ClearLatestByTaskID(taskID uint) error {
	if m.ClearLatestByTaskIDFunc != nil {
		return m.ClearLatestByTaskIDFunc(taskID)
	}
	return nil
}

func (m *mockDocumentRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	if m.GetByTaskIDFunc != nil {
		return m.GetByTaskIDFunc(taskID)
	}
	return nil, nil
}

func TestDocumentServiceCreateVersioned(t *testing.T) {
	var captured *model.Document
	repo := &mockDocumentRepo{
		CreateVersionedFunc: func(doc *model.Document) error {
			captured = doc
			doc.Version = 2
			doc.IsLatest = true
			return nil
		},
	}
	service := NewDocumentService(&config.Config{}, repo, nil)

	doc, err := service.Create(CreateDocumentRequest{
		RepositoryID: 1,
		TaskID:       2,
		Title:        "概览",
		Filename:     "overview.md",
		Content:      "content",
		SortOrder:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if captured == nil {
		t.Fatalf("expected CreateVersioned to be called")
	}
	if captured.RepositoryID != 1 || captured.TaskID != 2 {
		t.Fatalf("unexpected captured document: %+v", captured)
	}
	if doc.Version != 2 || !doc.IsLatest {
		t.Fatalf("expected versioned document, got version=%d isLatest=%v", doc.Version, doc.IsLatest)
	}
}

func TestDocumentServiceCreateVersionedError(t *testing.T) {
	repo := &mockDocumentRepo{
		CreateVersionedFunc: func(doc *model.Document) error {
			return errors.New("create failed")
		},
	}
	service := NewDocumentService(&config.Config{}, repo, nil)

	_, err := service.Create(CreateDocumentRequest{
		RepositoryID: 1,
		TaskID:       2,
		Title:        "概览",
		Filename:     "overview.md",
		Content:      "content",
		SortOrder:    1,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDocumentServiceGetVersions(t *testing.T) {
	repo := &mockDocumentRepo{
		GetFunc: func(id uint) (*model.Document, error) {
			return &model.Document{ID: id, TaskID: 9}, nil
		},
		GetByTaskIDFunc: func(taskID uint) ([]model.Document, error) {
			return []model.Document{
				{ID: 3, TaskID: taskID, Version: 2},
				{ID: 2, TaskID: taskID, Version: 1},
			}, nil
		},
	}
	service := NewDocumentService(&config.Config{}, repo, nil)

	versions, err := service.GetVersions(3)
	if err != nil {
		t.Fatalf("GetVersions() error = %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}
