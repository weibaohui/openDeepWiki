package service

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type mockTaskRepo struct {
	lastCreated *model.Task
	createErr   error
}

type mockTaskHelperRepoRepo struct {
	repos map[uint]*model.Repository
}

func (m *mockTaskHelperRepoRepo) Create(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	if repo.ID == 0 {
		repo.ID = uint(len(m.repos) + 1)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockTaskHelperRepoRepo) List() ([]model.Repository, error) {
	var out []model.Repository
	for _, repo := range m.repos {
		out = append(out, *repo)
	}
	return out, nil
}

func (m *mockTaskHelperRepoRepo) Get(id uint) (*model.Repository, error) {
	repo, ok := m.repos[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}
	return repo, nil
}

func (m *mockTaskHelperRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	return m.Get(id)
}

func (m *mockTaskHelperRepoRepo) Save(repo *model.Repository) error {
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

func (m *mockTaskHelperRepoRepo) Delete(id uint) error {
	delete(m.repos, id)
	return nil
}

func (m *mockTaskRepo) Create(task *model.Task) error {
	m.lastCreated = task
	if task.ID == 0 {
		task.ID = 1
	}
	return m.createErr
}

func (m *mockTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) GetByStatus(status string) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) Get(id uint) (*model.Task, error) {
	return nil, nil
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

func TestCreateIncrementalWriteTask(t *testing.T) {
	repo := &mockTaskRepo{}
	repoRepo := &mockTaskHelperRepoRepo{
		repos: map[uint]*model.Repository{
			12: {ID: 12, Name: "repo-12"},
		},
	}
	svc := &TaskService{taskRepo: repo, repoRepo: repoRepo}

	task, err := svc.CreateIncrementalWriteTask(context.Background(), 12, "增量分析", 5)
	if err != nil {
		t.Fatalf("CreateIncrementalWriteTask error: %v", err)
	}
	if task.RepositoryID != 12 {
		t.Fatalf("unexpected repository id: %d", task.RepositoryID)
	}
	if task.Title != "增量分析" {
		t.Fatalf("unexpected title: %s", task.Title)
	}
	if task.WriterName != domain.IncrementalWriter {
		t.Fatalf("unexpected writer: %s", task.WriterName)
	}
	if task.TaskType != domain.IncrementalWrite {
		t.Fatalf("unexpected task type: %s", task.TaskType)
	}
	if task.Status != string(statemachine.TaskStatusPending) {
		t.Fatalf("unexpected status: %s", task.Status)
	}
	if task.SortOrder != 5 {
		t.Fatalf("unexpected sort order: %d", task.SortOrder)
	}
	if repo.lastCreated == nil {
		t.Fatalf("task not created in repository")
	}
}
