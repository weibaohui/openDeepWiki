package service

import (
	"context"
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, nil)
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, dirMaker)
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

	service := NewRepositoryService(&config.Config{}, repoRepo, &mockTaskRepo{}, &mockDocumentRepo{}, nil, dirMaker)
	if _, err := service.AnalyzeDirectory(context.Background(), 11); err == nil {
		t.Fatalf("expected error for disallowed status")
	}

	select {
	case <-called:
		t.Fatalf("did not expect directory analysis to run")
	case <-time.After(100 * time.Millisecond):
	}
}
