package service

import (
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type mockRepoRepository struct {
	GetBasicFunc func(id uint) (*model.Repository, error)
}

func (m *mockRepoRepository) Create(repo *model.Repository) error { return nil }

func (m *mockRepoRepository) List() ([]model.Repository, error) { return nil, nil }

func (m *mockRepoRepository) Get(id uint) (*model.Repository, error) { return m.GetBasic(id) }

func (m *mockRepoRepository) GetBasic(id uint) (*model.Repository, error) {
	if m.GetBasicFunc != nil {
		return m.GetBasicFunc(id)
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepoRepository) Save(repo *model.Repository) error { return nil }

func (m *mockRepoRepository) Delete(id uint) error { return nil }

func TestPrepareAnalyzeRepositoryReady(t *testing.T) {
	repoRepo := &mockRepoRepository{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return &model.Repository{ID: id, Status: string(statemachine.RepoStatusReady)}, nil
		},
	}
	svc := &RepositoryService{repoRepo: repoRepo}

	repo, err := svc.prepareAnalyzeRepository(1, "目录分析")
	if err != nil {
		t.Fatalf("prepareAnalyzeRepository error: %v", err)
	}
	if repo.ID != 1 {
		t.Fatalf("unexpected repo id: %d", repo.ID)
	}
}

func TestPrepareAnalyzeRepositoryDisallow(t *testing.T) {
	repoRepo := &mockRepoRepository{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return &model.Repository{ID: id, Status: string(statemachine.RepoStatusCloning)}, nil
		},
	}
	svc := &RepositoryService{repoRepo: repoRepo}

	_, err := svc.prepareAnalyzeRepository(2, "目录分析")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "仓库状态不允许执行") {
		t.Fatalf("unexpected error: %v", err)
	}
}
