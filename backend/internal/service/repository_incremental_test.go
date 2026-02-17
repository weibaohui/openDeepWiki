package service

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockRepoRepo struct {
	repos map[uint]*model.Repository
	err   error
}

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

func (m *mockRepoRepo) Get(id uint) (*model.Repository, error) {
	return m.GetBasic(id)
}

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

func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockRepoRepo) Delete(id uint) error {
	delete(m.repos, id)
	return nil
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v, output=%s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output))
}

// TestUpdateRepositoryCloneInfoSuccess 验证更新仓库提交信息成功。
func TestUpdateRepositoryCloneInfoSuccess(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, CloneBranch: "main", CloneCommit: "abc"},
	}}
	svc := NewRepositoryService(&config.Config{}, repoRepo, nil, nil, nil, nil)

	if err := svc.UpdateRepositoryCloneInfo(context.Background(), 1, "dev", "def"); err != nil {
		t.Fatalf("UpdateRepositoryCloneInfo error: %v", err)
	}
	repo := repoRepo.repos[1]
	if repo.CloneBranch != "dev" || repo.CloneCommit != "def" {
		t.Fatalf("unexpected repo clone info: %+v", repo)
	}
}

// TestUpdateRepositoryCloneInfoEmptyCommit 验证提交为空时返回错误。
func TestUpdateRepositoryCloneInfoEmptyCommit(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, CloneBranch: "main", CloneCommit: "abc"},
	}}
	svc := NewRepositoryService(&config.Config{}, repoRepo, nil, nil, nil, nil)

	if err := svc.UpdateRepositoryCloneInfo(context.Background(), 1, "dev", ""); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
