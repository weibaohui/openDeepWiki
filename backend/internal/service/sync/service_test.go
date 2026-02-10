package syncservice

import (
	"errors"
	"testing"
	"time"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

type mockRepoRepo struct {
	repos map[uint]*model.Repository
	err   error
}

// Create 创建仓库
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

// List 列出仓库
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

// Get 获取仓库
func (m *mockRepoRepo) Get(id uint) (*model.Repository, error) {
	return m.GetBasic(id)
}

// GetBasic 获取仓库基础信息
func (m *mockRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.err != nil {
		return nil, m.err
	}
	repo, ok := m.repos[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return repo, nil
}

// Save 保存仓库
func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

// Delete 删除仓库
func (m *mockRepoRepo) Delete(id uint) error {
	delete(m.repos, id)
	return nil
}

type mockTaskRepo struct {
	tasks  map[uint]*model.Task
	nextID uint
	err    error
}

// Create 创建任务
func (m *mockTaskRepo) Create(task *model.Task) error {
	if m.err != nil {
		return m.err
	}
	if m.tasks == nil {
		m.tasks = make(map[uint]*model.Task)
	}
	if task.ID == 0 {
		m.nextID++
		task.ID = m.nextID
	}
	m.tasks[task.ID] = task
	return nil
}

// GetByRepository 按仓库获取任务
func (m *mockTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Task
	for _, task := range m.tasks {
		if task.RepositoryID == repoID {
			out = append(out, *task)
		}
	}
	return out, nil
}

// GetByStatus 按状态获取任务
func (m *mockTaskRepo) GetByStatus(status string) ([]model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Task
	for _, task := range m.tasks {
		if task.Status == status {
			out = append(out, *task)
		}
	}
	return out, nil
}

// Get 获取任务
func (m *mockTaskRepo) Get(id uint) (*model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return task, nil
}

// Save 保存任务
func (m *mockTaskRepo) Save(task *model.Task) error {
	if m.err != nil {
		return m.err
	}
	if m.tasks == nil {
		m.tasks = make(map[uint]*model.Task)
	}
	m.tasks[task.ID] = task
	return nil
}

// CleanupStuckTasks 清理卡住的任务
func (m *mockTaskRepo) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	return 0, m.err
}

// GetStuckTasks 获取卡住的任务
func (m *mockTaskRepo) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return nil, m.err
}

// DeleteByRepositoryID 删除仓库下的任务
func (m *mockTaskRepo) DeleteByRepositoryID(repoID uint) error {
	if m.err != nil {
		return m.err
	}
	for id, task := range m.tasks {
		if task.RepositoryID == repoID {
			delete(m.tasks, id)
		}
	}
	return nil
}

// Delete 删除任务
func (m *mockTaskRepo) Delete(id uint) error {
	if m.err != nil {
		return m.err
	}
	delete(m.tasks, id)
	return nil
}

// GetTaskStats 获取任务统计
func (m *mockTaskRepo) GetTaskStats(repoID uint) (map[string]int64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string]int64{}, nil
}

// GetActiveTasks 获取活跃任务
func (m *mockTaskRepo) GetActiveTasks() ([]model.Task, error) {
	return nil, m.err
}

// GetRecentTasks 获取最近任务
func (m *mockTaskRepo) GetRecentTasks(limit int) ([]model.Task, error) {
	return nil, m.err
}

type mockDocRepo struct {
	docs   map[uint]*model.Document
	nextID uint
	err    error
}

// Create 创建文档
func (m *mockDocRepo) Create(doc *model.Document) error {
	if m.err != nil {
		return m.err
	}
	if m.docs == nil {
		m.docs = make(map[uint]*model.Document)
	}
	if doc.ID == 0 {
		m.nextID++
		doc.ID = m.nextID
	}
	m.docs[doc.ID] = doc
	return nil
}

// GetByRepository 按仓库获取文档
func (m *mockDocRepo) GetByRepository(repoID uint) ([]model.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Document
	for _, doc := range m.docs {
		if doc.RepositoryID == repoID {
			out = append(out, *doc)
		}
	}
	return out, nil
}

// GetVersions 获取版本列表
func (m *mockDocRepo) GetVersions(repoID uint, title string) ([]model.Document, error) {
	return nil, m.err
}

// Get 获取文档
func (m *mockDocRepo) Get(id uint) (*model.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	doc, ok := m.docs[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return doc, nil
}

// Save 保存文档
func (m *mockDocRepo) Save(doc *model.Document) error {
	if m.err != nil {
		return m.err
	}
	if m.docs == nil {
		m.docs = make(map[uint]*model.Document)
	}
	m.docs[doc.ID] = doc
	return nil
}

// Delete 删除文档
func (m *mockDocRepo) Delete(id uint) error {
	if m.err != nil {
		return m.err
	}
	delete(m.docs, id)
	return nil
}

// DeleteByTaskID 删除任务下的文档
func (m *mockDocRepo) DeleteByTaskID(taskID uint) error {
	if m.err != nil {
		return m.err
	}
	for id, doc := range m.docs {
		if doc.TaskID == taskID {
			delete(m.docs, id)
		}
	}
	return nil
}

// DeleteByRepositoryID 删除仓库下的文档
func (m *mockDocRepo) DeleteByRepositoryID(repoID uint) error {
	if m.err != nil {
		return m.err
	}
	for id, doc := range m.docs {
		if doc.RepositoryID == repoID {
			delete(m.docs, id)
		}
	}
	return nil
}

// UpdateTaskID 更新文档任务ID
func (m *mockDocRepo) UpdateTaskID(docID uint, taskID uint) error {
	if m.err != nil {
		return m.err
	}
	doc, ok := m.docs[docID]
	if !ok {
		return repository.ErrNotFound
	}
	doc.TaskID = taskID
	return nil
}

// TransferLatest 转移最新版本标记
func (m *mockDocRepo) TransferLatest(oldDocID uint, newDocID uint) error {
	return m.err
}

// CreateVersioned 创建版本化文档
func (m *mockDocRepo) CreateVersioned(doc *model.Document) error {
	return m.Create(doc)
}

// GetLatestVersionByTaskID 获取最新版本号
func (m *mockDocRepo) GetLatestVersionByTaskID(taskID uint) (int, error) {
	return 0, m.err
}

// ClearLatestByTaskID 清理最新版本标记
func (m *mockDocRepo) ClearLatestByTaskID(taskID uint) error {
	return m.err
}

// GetByTaskID 按任务获取文档
func (m *mockDocRepo) GetByTaskID(taskID uint) ([]model.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Document
	for _, doc := range m.docs {
		if doc.TaskID == taskID {
			out = append(out, *doc)
		}
	}
	return out, nil
}

// TestServiceCreateTaskSuccess 验证创建任务成功
func TestServiceCreateTaskSuccess(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Name: "repo-1"},
	}}
	taskRepo := &mockTaskRepo{}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.TaskCreateRequest{
		RepositoryID: 1,
		Title:        "任务A",
		Status:       "completed",
		SortOrder:    2,
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}

	task, err := svc.CreateTask(nil, req)
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if task.ID == 0 {
		t.Fatalf("expected task id")
	}
	if task.RepositoryID != 1 || task.Title != "任务A" || task.Status != "completed" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

// TestServiceCreateTaskRepoNotFound 验证仓库不存在时返回错误
func TestServiceCreateTaskRepoNotFound(t *testing.T) {
	repoRepo := &mockRepoRepo{err: errors.New("not found")}
	taskRepo := &mockTaskRepo{}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.TaskCreateRequest{
		RepositoryID: 2,
		Title:        "任务B",
	}
	if _, err := svc.CreateTask(nil, req); err == nil {
		t.Fatalf("expected error")
	}
}

// TestServiceCreateDocumentSuccess 验证创建文档成功
func TestServiceCreateDocumentSuccess(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Name: "repo-1"},
	}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		10: {ID: 10, RepositoryID: 1, Title: "任务"},
	}}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.DocumentCreateRequest{
		RepositoryID: 1,
		TaskID:       10,
		Title:        "文档",
		Filename:     "doc.md",
		Content:      "内容",
		SortOrder:    1,
		Version:      1,
		IsLatest:     true,
		CreatedAt:    time.Now().Add(-time.Minute),
		UpdatedAt:    time.Now(),
	}

	doc, err := svc.CreateDocument(nil, req)
	if err != nil {
		t.Fatalf("CreateDocument error: %v", err)
	}
	if doc.ID == 0 || doc.TaskID != 10 {
		t.Fatalf("unexpected doc: %+v", doc)
	}
}

// TestServiceCreateDocumentRepoMismatch 验证仓库不匹配时返回错误
func TestServiceCreateDocumentRepoMismatch(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Name: "repo-1"},
	}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		10: {ID: 10, RepositoryID: 2, Title: "任务"},
	}}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.DocumentCreateRequest{
		RepositoryID: 1,
		TaskID:       10,
		Title:        "文档",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if _, err := svc.CreateDocument(nil, req); err == nil {
		t.Fatalf("expected error")
	}
}

// TestServiceUpdateTaskDocID 验证更新任务文档ID成功
func TestServiceUpdateTaskDocID(t *testing.T) {
	repoRepo := &mockRepoRepo{}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		7: {ID: 7, RepositoryID: 1, Title: "任务"},
	}}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	task, err := svc.UpdateTaskDocID(nil, 7, 99)
	if err != nil {
		t.Fatalf("UpdateTaskDocID error: %v", err)
	}
	if task.DocID != 99 {
		t.Fatalf("unexpected doc id: %d", task.DocID)
	}
}

// TestServiceCreateOrUpdateRepositoryCreate 验证新增仓库同步成功
func TestServiceCreateOrUpdateRepositoryCreate(t *testing.T) {
	repoRepo := &mockRepoRepo{}
	taskRepo := &mockTaskRepo{}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.RepositoryUpsertRequest{
		RepositoryID: 3,
		Name:         "repo-3",
		URL:          "https://example.com/repo-3",
		Status:       "ready",
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}

	repo, err := svc.CreateOrUpdateRepository(nil, req)
	if err != nil {
		t.Fatalf("CreateOrUpdateRepository error: %v", err)
	}
	if repo.ID != 3 || repo.Name != "repo-3" || repo.URL != "https://example.com/repo-3" {
		t.Fatalf("unexpected repo: %+v", repo)
	}
}

// TestServiceCreateOrUpdateRepositoryUpdate 验证更新仓库同步成功
func TestServiceCreateOrUpdateRepositoryUpdate(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		5: {ID: 5, Name: "old", URL: "https://example.com/old"},
	}}
	taskRepo := &mockTaskRepo{}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo)

	req := syncdto.RepositoryUpsertRequest{
		RepositoryID: 5,
		Name:         "new",
		URL:          "https://example.com/new",
		Status:       "ready",
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}

	repo, err := svc.CreateOrUpdateRepository(nil, req)
	if err != nil {
		t.Fatalf("CreateOrUpdateRepository error: %v", err)
	}
	if repo.ID != 5 || repo.Name != "new" || repo.URL != "https://example.com/new" {
		t.Fatalf("unexpected repo: %+v", repo)
	}
}
