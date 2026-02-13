package syncservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
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
		return nil, domain.ErrRecordNotFound
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
		return nil, domain.ErrRecordNotFound
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
		return nil, domain.ErrRecordNotFound
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

// GetTokenUsageByDocID 根据 document_id 获取 Token 用量数据
func (m *mockDocRepo) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	return nil, nil
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
		return domain.ErrRecordNotFound
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

type mockTaskUsageRepo struct {
	usages map[uint]*model.TaskUsage
	err    error
}

// Create 新增任务用量记录
func (m *mockTaskUsageRepo) Create(ctx context.Context, usage *model.TaskUsage) error {
	if m.err != nil {
		return m.err
	}
	if m.usages == nil {
		m.usages = make(map[uint]*model.TaskUsage)
	}
	m.usages[usage.TaskID] = usage
	return nil
}

// GetByTaskID 根据 task_id 查询任务用量记录
func (m *mockTaskUsageRepo) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
	if m.err != nil {
		return nil, m.err
	}
	usage, ok := m.usages[taskID]
	if !ok {
		return nil, nil
	}
	return usage, nil
}

// Upsert 根据 task_id 插入或更新任务用量记录
func (m *mockTaskUsageRepo) Upsert(ctx context.Context, usage *model.TaskUsage) error {
	if m.err != nil {
		return m.err
	}
	if m.usages == nil {
		m.usages = make(map[uint]*model.TaskUsage)
	}
	// 删除旧记录并插入新记录（覆盖逻辑）
	delete(m.usages, usage.TaskID)
	m.usages[usage.TaskID] = usage
	return nil
}

// TestServiceCreateTaskSuccess 验证创建任务成功
func TestServiceCreateTaskSuccess(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Name: "repo-1"},
	}}
	taskRepo := &mockTaskRepo{}
	docRepo := &mockDocRepo{}
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

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

func TestServiceClearRepositoryData(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		9: {ID: 9, Name: "repo-9"},
	}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		1: {ID: 1, RepositoryID: 9, Title: "任务A"},
		2: {ID: 2, RepositoryID: 10, Title: "任务B"},
	}}
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{
		3: {ID: 3, RepositoryID: 9, TaskID: 1, Title: "文档A"},
		4: {ID: 4, RepositoryID: 10, TaskID: 2, Title: "文档B"},
	}}
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

	if err := svc.ClearRepositoryData(nil, 9); err != nil {
		t.Fatalf("ClearRepositoryData error: %v", err)
	}
	if len(taskRepo.tasks) != 1 {
		t.Fatalf("unexpected task count: %d", len(taskRepo.tasks))
	}
	if len(docRepo.docs) != 1 {
		t.Fatalf("unexpected doc count: %d", len(docRepo.docs))
	}
	if taskRepo.tasks[2] == nil || docRepo.docs[4] == nil {
		t.Fatalf("unexpected remaining data")
	}
}

func TestNormalizeDocumentIDs(t *testing.T) {
	out := normalizeDocumentIDs([]uint{0, 2, 2, 3, 0, 1})
	if len(out) != 3 {
		t.Fatalf("unexpected length: %d", len(out))
	}
	if out[0] != 2 || out[1] != 3 || out[2] != 1 {
		t.Fatalf("unexpected order: %v", out)
	}
}

func TestFilterTasksByID(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "任务1"},
		{ID: 2, Title: "任务2"},
		{ID: 3, Title: "任务3"},
	}
	taskIDs := map[uint]struct{}{
		1: {},
		3: {},
	}
	filtered := filterTasksByID(tasks, taskIDs)
	if len(filtered) != 2 {
		t.Fatalf("unexpected length: %d", len(filtered))
	}
	if filtered[0].ID != 1 || filtered[1].ID != 3 {
		t.Fatalf("unexpected tasks: %v", filtered)
	}
}

func TestFilterDocumentsByID(t *testing.T) {
	docs := []model.Document{
		{ID: 10, Title: "文档1"},
		{ID: 11, Title: "文档2"},
		{ID: 12, Title: "文档3"},
	}
	docIDs := map[uint]struct{}{
		11: {},
	}
	filtered := filterDocumentsByID(docs, docIDs)
	if len(filtered) != 1 {
		t.Fatalf("unexpected length: %d", len(filtered))
	}
	if filtered[0].ID != 11 {
		t.Fatalf("unexpected docs: %v", filtered)
	}
}

func TestSelectLatestDocument(t *testing.T) {
	docs := []model.Document{
		{ID: 1, Version: 1},
		{ID: 3, Version: 2},
		{ID: 2, Version: 2},
	}
	latest := selectLatestDocument(docs)
	if latest == nil {
		t.Fatalf("expected latest document")
	}
	if latest.ID != 3 {
		t.Fatalf("unexpected latest: %v", latest.ID)
	}
}

func TestCollectTaskIDsByDocuments(t *testing.T) {
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{
		1: {ID: 1, RepositoryID: 7, TaskID: 10},
		2: {ID: 2, RepositoryID: 7, TaskID: 11},
	}}
	svc := New(&mockRepoRepo{}, &mockTaskRepo{}, docRepo, &mockTaskUsageRepo{})
	taskIDs, err := svc.collectTaskIDsByDocuments(nil, 7, []uint{1, 2})
	if err != nil {
		t.Fatalf("collectTaskIDsByDocuments error: %v", err)
	}
	if len(taskIDs) != 2 {
		t.Fatalf("unexpected taskIDs size: %d", len(taskIDs))
	}
	if _, ok := taskIDs[10]; !ok {
		t.Fatalf("missing task id 10")
	}
	if _, ok := taskIDs[11]; !ok {
		t.Fatalf("missing task id 11")
	}
}

func TestCollectTaskIDsByDocumentsMismatch(t *testing.T) {
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{
		1: {ID: 1, RepositoryID: 8, TaskID: 10},
	}}
	svc := New(&mockRepoRepo{}, &mockTaskRepo{}, docRepo, &mockTaskUsageRepo{})
	if _, err := svc.collectTaskIDsByDocuments(nil, 7, []uint{1}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildPullExportDataWithFilter(t *testing.T) {
	now := time.Now()
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Name: "repo-1", URL: "https://example.com/repo-1", CreatedAt: now, UpdatedAt: now},
	}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		10: {ID: 10, RepositoryID: 1, Title: "任务A", Status: "completed", CreatedAt: now, UpdatedAt: now},
		11: {ID: 11, RepositoryID: 1, Title: "任务B", Status: "running", CreatedAt: now, UpdatedAt: now},
	}}
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{
		100: {ID: 100, RepositoryID: 1, TaskID: 10, Title: "文档A", CreatedAt: now, UpdatedAt: now},
		101: {ID: 101, RepositoryID: 1, TaskID: 11, Title: "文档B", CreatedAt: now, UpdatedAt: now},
	}}
	taskUsageRepo := &mockTaskUsageRepo{usages: map[uint]*model.TaskUsage{
		10: {TaskID: 10, APIKeyName: "gpt-4", PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30, CreatedAt: now},
	}}
	svc := New(repoRepo, taskRepo, docRepo, taskUsageRepo)

	export, err := svc.BuildPullExportData(nil, 1, []uint{100})
	if err != nil {
		t.Fatalf("BuildPullExportData error: %v", err)
	}
	if export.Repository.RepositoryID != 1 {
		t.Fatalf("unexpected repository id: %d", export.Repository.RepositoryID)
	}
	if len(export.Tasks) != 1 || export.Tasks[0].TaskID != 10 {
		t.Fatalf("unexpected tasks: %+v", export.Tasks)
	}
	if len(export.Documents) != 1 || export.Documents[0].DocumentID != 100 {
		t.Fatalf("unexpected documents: %+v", export.Documents)
	}
	if len(export.TaskUsages) != 1 || export.TaskUsages[0].TaskID != 10 {
		t.Fatalf("unexpected task usages: %+v", export.TaskUsages)
	}
}

func TestListDocuments(t *testing.T) {
	now := time.Now()
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		2: {ID: 2, Name: "repo-2"},
	}}
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		21: {ID: 21, RepositoryID: 2, Status: "completed"},
	}}
	docRepo := &mockDocRepo{docs: map[uint]*model.Document{
		201: {ID: 201, RepositoryID: 2, TaskID: 21, Title: "文档C", CreatedAt: now},
	}}
	svc := New(repoRepo, taskRepo, docRepo, &mockTaskUsageRepo{})

	items, err := svc.ListDocuments(nil, 2)
	if err != nil {
		t.Fatalf("ListDocuments error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected items: %+v", items)
	}
	if items[0].DocumentID != 201 || items[0].Status != "completed" {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}
