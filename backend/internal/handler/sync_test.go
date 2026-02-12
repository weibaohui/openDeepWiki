package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	syncservice "github.com/weibaohui/opendeepwiki/backend/internal/service/sync"
)

type mockSyncRepoRepo struct {
	repos map[uint]*model.Repository
	err   error
}

// Create 创建仓库
func (m *mockSyncRepoRepo) Create(repo *model.Repository) error {
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
func (m *mockSyncRepoRepo) List() ([]model.Repository, error) {
	var out []model.Repository
	for _, repo := range m.repos {
		out = append(out, *repo)
	}
	return out, m.err
}

// Get 获取仓库
func (m *mockSyncRepoRepo) Get(id uint) (*model.Repository, error) {
	return m.GetBasic(id)
}

// GetBasic 获取仓库基础信息
func (m *mockSyncRepoRepo) GetBasic(id uint) (*model.Repository, error) {
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
func (m *mockSyncRepoRepo) Save(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

// Delete 删除仓库
func (m *mockSyncRepoRepo) Delete(id uint) error {
	delete(m.repos, id)
	return nil
}

type mockSyncTaskRepo struct{}

// Create 创建任务
func (m *mockSyncTaskRepo) Create(task *model.Task) error { return nil }

// GetByRepository 按仓库获取任务
func (m *mockSyncTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	return nil, nil
}

// GetByStatus 按状态获取任务
func (m *mockSyncTaskRepo) GetByStatus(status string) ([]model.Task, error) { return nil, nil }

// Get 获取任务
func (m *mockSyncTaskRepo) Get(id uint) (*model.Task, error) { return nil, repository.ErrNotFound }

// Save 保存任务
func (m *mockSyncTaskRepo) Save(task *model.Task) error { return nil }

// CleanupStuckTasks 清理卡住的任务
func (m *mockSyncTaskRepo) CleanupStuckTasks(timeout time.Duration) (int64, error) { return 0, nil }

// GetStuckTasks 获取卡住的任务
func (m *mockSyncTaskRepo) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	return nil, nil
}

// DeleteByRepositoryID 删除仓库下的任务
func (m *mockSyncTaskRepo) DeleteByRepositoryID(repoID uint) error { return nil }

// Delete 删除任务
func (m *mockSyncTaskRepo) Delete(id uint) error { return nil }

// GetTaskStats 获取任务统计
func (m *mockSyncTaskRepo) GetTaskStats(repoID uint) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// GetActiveTasks 获取活跃任务
func (m *mockSyncTaskRepo) GetActiveTasks() ([]model.Task, error) { return nil, nil }

// GetRecentTasks 获取最近任务
func (m *mockSyncTaskRepo) GetRecentTasks(limit int) ([]model.Task, error) { return nil, nil }

type mockSyncDocRepo struct{}

// Create 创建文档
func (m *mockSyncDocRepo) Create(doc *model.Document) error { return nil }

// GetByRepository 按仓库获取文档
func (m *mockSyncDocRepo) GetByRepository(repoID uint) ([]model.Document, error) { return nil, nil }

// GetVersions 获取版本列表
func (m *mockSyncDocRepo) GetVersions(repoID uint, title string) ([]model.Document, error) {
	return nil, nil
}

// Get 获取文档
func (m *mockSyncDocRepo) Get(id uint) (*model.Document, error) { return nil, repository.ErrNotFound }

// GetTokenUsageByDocID 根据 document_id 获取 Token 用量数据
func (m *mockSyncDocRepo) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	return nil, nil
}

// Save 保存文档
func (m *mockSyncDocRepo) Save(doc *model.Document) error { return nil }

// Delete 删除文档
func (m *mockSyncDocRepo) Delete(id uint) error { return nil }

// DeleteByTaskID 删除任务下文档
func (m *mockSyncDocRepo) DeleteByTaskID(taskID uint) error { return nil }

// DeleteByRepositoryID 删除仓库下文档
func (m *mockSyncDocRepo) DeleteByRepositoryID(repoID uint) error { return nil }

// UpdateTaskID 更新任务ID
func (m *mockSyncDocRepo) UpdateTaskID(docID uint, taskID uint) error { return nil }

// TransferLatest 转移最新版本标记
func (m *mockSyncDocRepo) TransferLatest(oldDocID uint, newDocID uint) error { return nil }

// CreateVersioned 创建版本化文档
func (m *mockSyncDocRepo) CreateVersioned(doc *model.Document) error { return nil }

// GetLatestVersionByTaskID 获取最新版本号
func (m *mockSyncDocRepo) GetLatestVersionByTaskID(taskID uint) (int, error) { return 0, nil }

// ClearLatestByTaskID 清理最新标记
func (m *mockSyncDocRepo) ClearLatestByTaskID(taskID uint) error { return nil }

// GetByTaskID 按任务获取文档
func (m *mockSyncDocRepo) GetByTaskID(taskID uint) ([]model.Document, error) { return nil, nil }

type mockSyncTaskUsageRepo struct{}

// Create 新增任务用量记录
func (m *mockSyncTaskUsageRepo) Create(ctx context.Context, usage *model.TaskUsage) error { return nil }

// GetByTaskID 根据 task_id 查询任务用量记录
func (m *mockSyncTaskUsageRepo) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) { return nil, nil }

// Upsert 根据 task_id 插入或更新任务用量记录
func (m *mockSyncTaskUsageRepo) Upsert(ctx context.Context, usage *model.TaskUsage) error { return nil }

// TestSyncHandlerRepositoryUpsert 验证仓库同步接口创建仓库成功
func TestSyncHandlerRepositoryUpsert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repoRepo := &mockSyncRepoRepo{}
	taskRepo := &mockSyncTaskRepo{}
	docRepo := &mockSyncDocRepo{}
	taskUsageRepo := &mockSyncTaskUsageRepo{}
	svc := syncservice.New(repoRepo, taskRepo, docRepo, taskUsageRepo)
	handler := NewSyncHandler(svc)
	router := gin.New()
	router.POST("/sync/repository-upsert", handler.RepositoryUpsert)

	payload := syncdto.RepositoryUpsertRequest{
		RepositoryID: 11,
		Name:         "repo-11",
		URL:          "https://example.com/repo-11",
		Status:       "ready",
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload error: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/sync/repository-upsert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp syncdto.RepositoryUpsertResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if resp.Data.RepositoryID != 11 || resp.Data.Name != "repo-11" {
		t.Fatalf("unexpected response: %+v", resp.Data)
	}
}

func TestSyncHandlerRepositoryClear(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repoRepo := &mockSyncRepoRepo{repos: map[uint]*model.Repository{
		8: {ID: 8, Name: "repo-8"},
	}}
	taskRepo := &mockSyncTaskRepo{}
	docRepo := &mockSyncDocRepo{}
	taskUsageRepo := &mockSyncTaskUsageRepo{}
	svc := syncservice.New(repoRepo, taskRepo, docRepo, taskUsageRepo)
	handler := NewSyncHandler(svc)
	router := gin.New()
	router.POST("/sync/repository-clear", handler.RepositoryClear)

	payload := syncdto.RepositoryClearRequest{RepositoryID: 8}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload error: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/sync/repository-clear", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp syncdto.RepositoryClearResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	if resp.Data.RepositoryID != 8 {
		t.Fatalf("unexpected response: %+v", resp.Data)
	}
}
