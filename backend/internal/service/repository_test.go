package service

import (
	"os"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

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
