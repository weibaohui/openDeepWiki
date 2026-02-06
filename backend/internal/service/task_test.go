package service

import (
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockRepoRepo struct {
	CreateFunc   func(repo *model.Repository) error
	ListFunc     func() ([]model.Repository, error)
	GetFunc      func(id uint) (*model.Repository, error)
	GetBasicFunc func(id uint) (*model.Repository, error)
	SaveFunc     func(repo *model.Repository) error
	DeleteFunc   func(id uint) error
}

func (m *mockRepoRepo) Create(repo *model.Repository) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(repo)
	}
	return nil
}

func (m *mockRepoRepo) List() ([]model.Repository, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil, nil
}

func (m *mockRepoRepo) Get(id uint) (*model.Repository, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *mockRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	if m.GetBasicFunc != nil {
		return m.GetBasicFunc(id)
	}
	return nil, nil
}

func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(repo)
	}
	return nil
}

func (m *mockRepoRepo) Delete(id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

type mockTaskRepo struct {
	CreateFunc               func(task *model.Task) error
	GetByRepositoryFunc      func(repoID uint) ([]model.Task, error)
	GetFunc                  func(id uint) (*model.Task, error)
	SaveFunc                 func(task *model.Task) error
	CleanupStuckTasksFunc    func(timeout time.Duration) (int64, error)
	GetStuckTasksFunc        func(timeout time.Duration) ([]model.Task, error)
	DeleteByRepositoryIDFunc func(repoID uint) error
	DeleteFunc               func(id uint) error
	GetTaskStatsFunc         func(repoID uint) (map[string]int64, error)
}

func (m *mockTaskRepo) Create(task *model.Task) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(task)
	}
	return nil
}

func (m *mockTaskRepo) GetByRepository(repoID uint) ([]model.Task, error) {
	if m.GetByRepositoryFunc != nil {
		return m.GetByRepositoryFunc(repoID)
	}
	return nil, nil
}

func (m *mockTaskRepo) Get(id uint) (*model.Task, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return nil, nil
}

func (m *mockTaskRepo) Save(task *model.Task) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(task)
	}
	return nil
}

func (m *mockTaskRepo) CleanupStuckTasks(timeout time.Duration) (int64, error) {
	if m.CleanupStuckTasksFunc != nil {
		return m.CleanupStuckTasksFunc(timeout)
	}
	return 0, nil
}

func (m *mockTaskRepo) GetStuckTasks(timeout time.Duration) ([]model.Task, error) {
	if m.GetStuckTasksFunc != nil {
		return m.GetStuckTasksFunc(timeout)
	}
	return nil, nil
}

func (m *mockTaskRepo) DeleteByRepositoryID(repoID uint) error {
	if m.DeleteByRepositoryIDFunc != nil {
		return m.DeleteByRepositoryIDFunc(repoID)
	}
	return nil
}

func (m *mockTaskRepo) Delete(id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *mockTaskRepo) GetTaskStats(repoID uint) (map[string]int64, error) {
	if m.GetTaskStatsFunc != nil {
		return m.GetTaskStatsFunc(repoID)
	}
	return nil, nil
}

func TestTaskServiceResetKeepsHistory(t *testing.T) {
	now := time.Now()
	task := &model.Task{
		ID:           1,
		RepositoryID: 2,
		Status:       "failed",
		ErrorMsg:     "error",
		StartedAt:    &now,
		CompletedAt:  &now,
	}
	var saved *model.Task
	repo := &model.Repository{ID: 2, Status: "ready"}
	taskRepo := &mockTaskRepo{
		GetFunc: func(id uint) (*model.Task, error) {
			return task, nil
		},
		GetByRepositoryFunc: func(repoID uint) ([]model.Task, error) {
			return []model.Task{*task}, nil
		},
		SaveFunc: func(t *model.Task) error {
			saved = t
			return nil
		},
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		SaveFunc: func(r *model.Repository) error {
			repo = r
			return nil
		},
	}
	service := NewTaskService(&config.Config{}, taskRepo, repoRepo, &DocumentService{}, nil)

	if err := service.Reset(1); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if saved == nil {
		t.Fatalf("expected task to be saved")
	}
	if saved.Status != "pending" || saved.ErrorMsg != "" || saved.StartedAt != nil || saved.CompletedAt != nil {
		t.Fatalf("unexpected task after reset: %+v", saved)
	}
}

func TestTaskServiceForceResetKeepsHistory(t *testing.T) {
	now := time.Now()
	task := &model.Task{
		ID:           2,
		RepositoryID: 3,
		Status:       "running",
		ErrorMsg:     "error",
		StartedAt:    &now,
	}
	var saved *model.Task
	repo := &model.Repository{ID: 3, Status: "ready"}
	taskRepo := &mockTaskRepo{
		GetFunc: func(id uint) (*model.Task, error) {
			return task, nil
		},
		GetByRepositoryFunc: func(repoID uint) ([]model.Task, error) {
			return []model.Task{*task}, nil
		},
		SaveFunc: func(t *model.Task) error {
			saved = t
			return nil
		},
	}
	repoRepo := &mockRepoRepo{
		GetBasicFunc: func(id uint) (*model.Repository, error) {
			return repo, nil
		},
		SaveFunc: func(r *model.Repository) error {
			repo = r
			return nil
		},
	}
	service := NewTaskService(&config.Config{}, taskRepo, repoRepo, &DocumentService{}, nil)

	if err := service.ForceReset(2); err != nil {
		t.Fatalf("ForceReset() error = %v", err)
	}
	if saved == nil {
		t.Fatalf("expected task to be saved")
	}
	if saved.Status != "pending" || saved.ErrorMsg != "" || saved.StartedAt != nil || saved.CompletedAt != nil {
		t.Fatalf("unexpected task after force reset: %+v", saved)
	}
}
