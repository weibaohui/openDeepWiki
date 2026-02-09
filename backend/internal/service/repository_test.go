package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type mockDirMakerService struct {
	CreateDirsFunc func(ctx context.Context, repo *model.Repository) ([]*model.Task, error)
}

func (m *mockDirMakerService) CreateDirs(ctx context.Context, repo *model.Repository) ([]*model.Task, error) {
	if m.CreateDirsFunc != nil {
		return m.CreateDirsFunc(ctx, repo)
	}
	return nil, nil
}

type mockDatabaseModelParser struct {
	GenerateFunc func(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error)
}

func (m *mockDatabaseModelParser) Generate(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, localPath, title, repoID, taskID)
	}
	return "", nil
}

type mockAPIAnalyzer struct {
	GenerateFunc func(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error)
}

// Generate 模拟API接口分析输出。
func (m *mockAPIAnalyzer) Generate(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, localPath, title, repoID, taskID)
	}
	return "", nil
}

func TestRepositoryServiceCreateInvalidURL(t *testing.T) {
	repoRepo := &mockRepoRepo{
		ListFunc: func() ([]model.Repository, error) {
			return nil, nil
		},
		CreateFunc: func(repo *model.Repository) error {
			t.Fatalf("unexpected create called")
			return nil
		},
	}
	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	_, err := service.Create(CreateRepoRequest{URL: "https://github.com/owner/repo/blob/main/README.md"})
	if !errors.Is(err, ErrInvalidRepositoryURL) {
		t.Fatalf("expected invalid url error, got %v", err)
	}
}

func TestRepositoryServiceCreateDuplicateURL(t *testing.T) {
	repoRepo := &mockRepoRepo{
		ListFunc: func() ([]model.Repository, error) {
			return []model.Repository{{ID: 1, URL: "https://github.com/Owner/Repo?tab=readme"}}, nil
		},
		CreateFunc: func(repo *model.Repository) error {
			t.Fatalf("unexpected create called")
			return nil
		},
	}
	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	_, err := service.Create(CreateRepoRequest{URL: "https://github.com/owner/repo"})
	if !errors.Is(err, ErrRepositoryAlreadyExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestRepositoryServicePurgeLocalDirSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "purge-local-dir")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        1,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: tempDir,
	}
	saveCalled := false
	savedLocalPath := "non-empty"

	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		SaveFunc: func(updated *model.Repository) error {
			saveCalled = true
			savedLocalPath = updated.LocalPath
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	if err := service.PurgeLocalDir(1); err != nil {
		t.Fatalf("PurgeLocalDir error: %v", err)
	}

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("expected local dir removed, stat err=%v", err)
	}

	if !saveCalled {
		t.Fatalf("expected Save to be called")
	}
	if savedLocalPath != "" {
		t.Fatalf("expected LocalPath cleared, got %s", savedLocalPath)
	}
}

func TestRepositoryServicePurgeLocalDirDisallowedStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "purge-local-dir")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        2,
		Status:    string(statemachine.RepoStatusCloning),
		LocalPath: tempDir,
	}

	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	if err := service.PurgeLocalDir(2); err == nil {
		t.Fatalf("expected error for cloning status")
	}

	if _, err := os.Stat(tempDir); err != nil {
		t.Fatalf("expected local dir exists, stat err=%v", err)
	}
}

func TestRepositoryServicePurgeLocalDirEmptyPath(t *testing.T) {
	repo := &model.Repository{
		ID:        3,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: "",
	}
	saveCalled := false

	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		SaveFunc: func(updated *model.Repository) error {
			saveCalled = true
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	if err := service.PurgeLocalDir(3); err != nil {
		t.Fatalf("PurgeLocalDir error: %v", err)
	}
	if saveCalled {
		t.Fatalf("did not expect Save to be called")
	}
}

func TestRepositoryServiceAnalyzeDirectoryAsync(t *testing.T) {
	repo := &model.Repository{
		ID:     10,
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, dirMaker, nil, nil, nil)
	tasks, err := service.AnalyzeDirectory(context.Background(), 10)
	if err != nil {
		t.Fatalf("AnalyzeDirectory error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected empty tasks response, got %d", len(tasks))
	}

	select {
	case <-called:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async directory analysis to be triggered")
	}
}

func TestRepositoryServiceAnalyzeDirectoryDisallowedStatus(t *testing.T) {
	repo := &model.Repository{
		ID:     11,
		Status: string(statemachine.RepoStatusCloning),
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, dirMaker, nil, nil, nil)
	if _, err := service.AnalyzeDirectory(context.Background(), 11); err == nil {
		t.Fatalf("expected error for disallowed status")
	}

	select {
	case <-called:
		t.Fatalf("did not expect directory analysis to run")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRepositoryServiceAnalyzeDatabaseModelAsync(t *testing.T) {
	repo := &model.Repository{
		ID:        12,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: "/tmp/repo",
	}
	triggered := make(chan struct{}, 1)
	parser := &mockDatabaseModelParser{
		GenerateFunc: func(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
			triggered <- struct{}{}
			return "# 数据库模型\n", nil
		},
	}
	taskRepo := &mockTaskRepo{
		CreateFunc: func(task *model.Task) error {
			task.ID = 20
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		CreateVersionedFunc: func(doc *model.Document) error {
			return nil
		},
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	docService := NewDocumentService(&config.Config{}, docRepo, repoRepo, nil)
	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, &mockDocumentRepo{}, nil, nil, docService, parser, nil)
	task, err := service.AnalyzeDatabaseModel(context.Background(), 12)
	if err != nil {
		t.Fatalf("AnalyzeDatabaseModel error: %v", err)
	}
	if task == nil || task.Type != "db-model" {
		t.Fatalf("unexpected task: %+v", task)
	}
	select {
	case <-triggered:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async database model analysis to be triggered")
	}
}

func TestRepositoryServiceAnalyzeDatabaseModelFailed(t *testing.T) {
	repo := &model.Repository{
		ID:        14,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: "/tmp/repo",
	}
	failedCh := make(chan model.Task, 1)
	parser := &mockDatabaseModelParser{
		GenerateFunc: func(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error) {
			return "", errors.New("解析失败")
		},
	}
	taskRepo := &mockTaskRepo{
		CreateFunc: func(task *model.Task) error {
			task.ID = 21
			return nil
		},
		SaveFunc: func(task *model.Task) error {
			if task.Status == string(statemachine.TaskStatusFailed) {
				failedCh <- *task
			}
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		CreateVersionedFunc: func(doc *model.Document) error {
			return nil
		},
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	docService := NewDocumentService(&config.Config{}, docRepo, repoRepo, nil)
	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, &mockDocumentRepo{}, nil, nil, docService, parser, nil)
	task, err := service.AnalyzeDatabaseModel(context.Background(), 14)
	if err != nil {
		t.Fatalf("AnalyzeDatabaseModel error: %v", err)
	}
	if task == nil || task.Type != "db-model" {
		t.Fatalf("unexpected task: %+v", task)
	}
	select {
	case failedTask := <-failedCh:
		if failedTask.CompletedAt == nil {
			t.Fatalf("expected completed time to be set")
		}
		if failedTask.StartedAt == nil {
			t.Fatalf("expected started time to be set")
		}
		if failedTask.ErrorMsg == "" || failedTask.ErrorMsg == "解析失败" {
			t.Fatalf("expected error message to be wrapped, got %s", failedTask.ErrorMsg)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected async database model analysis to fail")
	}
}

func TestRepositoryServiceAnalyzeDatabaseModelDisallowedStatus(t *testing.T) {
	repo := &model.Repository{
		ID:     13,
		Status: string(statemachine.RepoStatusCloning),
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	parser := &mockDatabaseModelParser{}
	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, parser, nil)
	if _, err := service.AnalyzeDatabaseModel(context.Background(), 13); err == nil {
		t.Fatalf("expected error for disallowed status")
	}
}

func TestRepositoryServiceDeleteCompletedStatus(t *testing.T) {
	repo := &model.Repository{
		ID:     100,
		Status: string(statemachine.RepoStatusCompleted),
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	err := service.Delete(100)
	if !errors.Is(err, ErrCannotDeleteRepoInvalidStatus) {
		t.Fatalf("expected ErrCannotDeleteRepoInvalidStatus, got %v", err)
	}
}

func TestRepositoryServiceDeleteAnalyzingStatus(t *testing.T) {
	repo := &model.Repository{
		ID:     101,
		Status: string(statemachine.RepoStatusAnalyzing),
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
	}
	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil, nil, nil, nil)
	err := service.Delete(101)
	if !errors.Is(err, ErrCannotDeleteRepoInvalidStatus) {
		t.Fatalf("expected ErrCannotDeleteRepoInvalidStatus, got %v", err)
	}
}

func TestRepositoryServiceDeletePendingStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "delete-repo")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        102,
		Status:    string(statemachine.RepoStatusPending),
		LocalPath: tempDir,
	}
	deleteCalled := false
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		DeleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}
	taskRepo := &mockTaskRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, docRepo, nil, nil, nil, nil, nil)
	err = service.Delete(102)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected repoRepo.Delete to be called")
	}
}

func TestRepositoryServiceDeleteCloningStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "delete-repo")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        103,
		Status:    string(statemachine.RepoStatusCloning),
		LocalPath: tempDir,
	}
	deleteCalled := false
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		DeleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}
	taskRepo := &mockTaskRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, docRepo, nil, nil, nil, nil, nil)
	err = service.Delete(103)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected repoRepo.Delete to be called")
	}
}

func TestRepositoryServiceDeleteReadyStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "delete-repo")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        104,
		Status:    string(statemachine.RepoStatusReady),
		LocalPath: tempDir,
	}
	deleteCalled := false
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		DeleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}
	taskRepo := &mockTaskRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, docRepo, nil, nil, nil, nil, nil)
	err = service.Delete(104)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected repoRepo.Delete to be called")
	}
}

func TestRepositoryServiceDeleteErrorStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "delete-repo")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := &model.Repository{
		ID:        105,
		Status:    string(statemachine.RepoStatusError),
		LocalPath: tempDir,
	}
	deleteCalled := false
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		DeleteFunc: func(id uint) error {
			deleteCalled = true
			return nil
		},
	}
	taskRepo := &mockTaskRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}
	docRepo := &mockDocumentRepo{
		DeleteByRepositoryIDFunc: func(repoID uint) error {
			return nil
		},
	}

	service := NewRepositoryService(&config.Config{}, repoRepo, taskRepo, docRepo, nil, nil, nil, nil, nil)
	err = service.Delete(105)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("expected repoRepo.Delete to be called")
	}
}
