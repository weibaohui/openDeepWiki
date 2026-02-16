package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

func TestRepositoryServiceIncrementalAnalysisSuccess(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	remotePath := filepath.Join(tempDir, "remote.git")
	runGit(t, "", "init", "--bare", remotePath)

	sourcePath := filepath.Join(tempDir, "source")
	runGit(t, "", "clone", remotePath, sourcePath)
	runGit(t, sourcePath, "config", "user.email", "test@example.com")
	runGit(t, sourcePath, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(sourcePath, "README.md"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	runGit(t, sourcePath, "add", ".")
	runGit(t, sourcePath, "commit", "-m", "init")
	baseCommit := runGit(t, sourcePath, "rev-parse", "HEAD")
	runGit(t, sourcePath, "push", "origin", "HEAD")

	targetPath := filepath.Join(tempDir, "target")
	runGit(t, "", "clone", remotePath, targetPath)

	if err := os.WriteFile(filepath.Join(sourcePath, "README.md"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	runGit(t, sourcePath, "add", ".")
	runGit(t, sourcePath, "commit", "-m", "update")
	runGit(t, sourcePath, "push", "origin", "HEAD")

	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, LocalPath: targetPath, CloneCommit: baseCommit},
	}}
	svc := &RepositoryService{repoRepo: repoRepo}

	if err := svc.IncrementalAnalysis(ctx, 1); err != nil {
		t.Fatalf("IncrementalAnalysis error: %v", err)
	}
}

// TestRepositoryServiceIncrementalAnalysisShallow 验证浅克隆场景下可补全历史并完成增量分析。
func TestRepositoryServiceIncrementalAnalysisShallow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	remotePath := filepath.Join(tempDir, "remote.git")
	runGit(t, "", "init", "--bare", remotePath)

	sourcePath := filepath.Join(tempDir, "source")
	runGit(t, "", "clone", remotePath, sourcePath)
	runGit(t, sourcePath, "config", "user.email", "test@example.com")
	runGit(t, sourcePath, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(sourcePath, "README.md"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	runGit(t, sourcePath, "add", ".")
	runGit(t, sourcePath, "commit", "-m", "init")
	baseCommit := runGit(t, sourcePath, "rev-parse", "HEAD")
	runGit(t, sourcePath, "push", "origin", "HEAD")

	if err := os.WriteFile(filepath.Join(sourcePath, "README.md"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	runGit(t, sourcePath, "add", ".")
	runGit(t, sourcePath, "commit", "-m", "update")
	runGit(t, sourcePath, "push", "origin", "HEAD")

	targetPath := filepath.Join(tempDir, "target")
	runGit(t, "", "clone", "--depth", "1", remotePath, targetPath)

	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, LocalPath: targetPath, CloneCommit: baseCommit},
	}}
	svc := &RepositoryService{repoRepo: repoRepo}

	if err := svc.IncrementalAnalysis(ctx, 1); err != nil {
		t.Fatalf("IncrementalAnalysis error: %v", err)
	}
}

func TestRepositoryServiceIncrementalAnalysisMissingBaseCommit(t *testing.T) {
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, LocalPath: os.TempDir()},
	}}
	svc := &RepositoryService{repoRepo: repoRepo}

	if err := svc.IncrementalAnalysis(context.Background(), 1); err == nil {
		t.Fatalf("expected error")
	}
}
