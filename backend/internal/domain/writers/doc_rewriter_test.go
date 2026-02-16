package writers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockDocRepo struct {
	docs map[uint]*model.Document
}

func (m *mockDocRepo) Create(doc *model.Document) error {
	if m.docs == nil {
		m.docs = make(map[uint]*model.Document)
	}
	m.docs[doc.ID] = doc
	return nil
}

func (m *mockDocRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockDocRepo) GetAllDocumentsTitleAndID(repoID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockDocRepo) GetVersions(repoID uint, title string) ([]model.Document, error) {
	return nil, nil
}

func (m *mockDocRepo) Get(id uint) (*model.Document, error) {
	doc, ok := m.docs[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	return doc, nil
}

func (m *mockDocRepo) Save(doc *model.Document) error {
	if m.docs == nil {
		m.docs = make(map[uint]*model.Document)
	}
	m.docs[doc.ID] = doc
	return nil
}

func (m *mockDocRepo) Delete(id uint) error {
	delete(m.docs, id)
	return nil
}

func (m *mockDocRepo) DeleteByTaskID(taskID uint) error {
	return nil
}

func (m *mockDocRepo) DeleteByRepositoryID(repoID uint) error {
	return nil
}

func (m *mockDocRepo) UpdateTaskID(docID uint, taskID uint) error {
	return nil
}

func (m *mockDocRepo) TransferLatest(oldDocID uint, newDocID uint) error {
	return nil
}

func (m *mockDocRepo) CreateVersioned(doc *model.Document) error {
	return nil
}

func (m *mockDocRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	return 0, nil
}

func (m *mockDocRepo) ClearLatestByTaskID(taskID uint) error {
	return nil
}

func (m *mockDocRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	return nil, nil
}

func (m *mockDocRepo) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	return nil, nil
}

type mockTaskRepo struct {
	tasks map[uint]*model.Task
}

func (m *mockTaskRepo) Create(task *model.Task) error {
	if m.tasks == nil {
		m.tasks = make(map[uint]*model.Task)
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) GetByStatus(status string) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) Get(id uint) (*model.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}
	return task, nil
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
	return map[string]int64{}, nil
}

func (m *mockTaskRepo) GetActiveTasks() ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) GetRecentTasks(limit int) ([]model.Task, error) {
	return nil, nil
}

// TestDocRewriter_GenerateEmptyGuide 验证重写指引为空时返回错误
func TestDocRewriter_GenerateEmptyGuide(t *testing.T) {
	doc := &model.Document{ID: 1, Title: "测试文档", Content: "原始内容"}
	task := &model.Task{ID: 10, DocID: 1, Outline: " "}
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{1: doc}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{10: task}}

	rewriter := &docRewriter{
		docRepo:  docRepo,
		taskRepo: taskRepo,
	}

	if _, err := rewriter.Generate(context.Background(), "", "", 10); err == nil {
		t.Fatalf("expected error for empty guide")
	}
	if docRepo.docs[1].Content != "原始内容" {
		t.Fatalf("unexpected content update")
	}
}
