package service

import (
	"context"
	"testing"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type mockTaskRepo struct {
	tasks map[uint]*model.Task
	err   error
}

// Create 创建任务
func (m *mockTaskRepo) Create(task *model.Task) error {
	if m.err != nil {
		return m.err
	}
	if m.tasks == nil {
		m.tasks = make(map[uint]*model.Task)
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
		return nil, repository.ErrNotFound
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

// DeleteByRepositoryID 删除仓库下任务
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

// GetTaskStats 获取任务状态统计
func (m *mockTaskRepo) GetTaskStats(repoID uint) (map[string]int64, error) {
	stats := make(map[string]int64)
	if m.err != nil {
		return stats, m.err
	}
	for _, task := range m.tasks {
		if task.RepositoryID == repoID {
			stats[task.Status]++
		}
	}
	return stats, nil
}

// GetActiveTasks 获取活跃任务
func (m *mockTaskRepo) GetActiveTasks() ([]model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Task
	for _, task := range m.tasks {
		if task.Status == string(statemachine.TaskStatusQueued) || task.Status == string(statemachine.TaskStatusRunning) {
			out = append(out, *task)
		}
	}
	return out, nil
}

// GetRecentTasks 获取最近任务
func (m *mockTaskRepo) GetRecentTasks(limit int) ([]model.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []model.Task
	for _, task := range m.tasks {
		out = append(out, *task)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type mockRepoRepo struct {
	repos map[uint]*model.Repository
	err   error
}

// Create 创建仓库
func (m *mockRepoRepo) Create(repo *model.Repository) error {
	if m.err != nil {
		return m.err
	}
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
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
	if m.err != nil {
		return nil, m.err
	}
	repo, ok := m.repos[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return repo, nil
}

// GetBasic 获取基础仓库信息
func (m *mockRepoRepo) GetBasic(id uint) (*model.Repository, error) {
	return m.Get(id)
}

// Save 保存仓库
func (m *mockRepoRepo) Save(repo *model.Repository) error {
	if m.err != nil {
		return m.err
	}
	if m.repos == nil {
		m.repos = make(map[uint]*model.Repository)
	}
	m.repos[repo.ID] = repo
	return nil
}

// Delete 删除仓库
func (m *mockRepoRepo) Delete(id uint) error {
	if m.err != nil {
		return m.err
	}
	delete(m.repos, id)
	return nil
}

type fakeExecutor struct{}

// ExecuteTask 伪执行任务
func (f *fakeExecutor) ExecuteTask(ctx context.Context, taskID uint) error {
	return nil
}

// TestTaskServiceEnqueueRunAfterNotSatisfied 验证依赖未满足时不入队
func TestTaskServiceEnqueueRunAfterNotSatisfied(t *testing.T) {
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		1: {ID: 1, RepositoryID: 1, Status: string(statemachine.TaskStatusFailed)},
		2: {ID: 2, RepositoryID: 1, Status: string(statemachine.TaskStatusPending), RunAfter: 1},
	}}
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Status: string(statemachine.RepoStatusReady)},
	}}
	svc := NewTaskService(&config.Config{}, taskRepo, repoRepo, nil)
	o, _ := orchestrator.NewOrchestrator(1, &fakeExecutor{})
	defer o.Stop()
	svc.SetOrchestrator(o)

	err := svc.Enqueue(2)
	if err != ErrRunAfterNotSatisfied {
		t.Fatalf("expected ErrRunAfterNotSatisfied, got %v", err)
	}
	if taskRepo.tasks[2].Status != string(statemachine.TaskStatusPending) {
		t.Fatalf("task status should remain pending, got %s", taskRepo.tasks[2].Status)
	}
	if got := o.GetQueueStatus().QueueLength; got != 0 {
		t.Fatalf("queue length should be 0, got %d", got)
	}
}

// TestTaskServiceEnqueueRunAfterSatisfied 验证依赖满足时可入队
func TestTaskServiceEnqueueRunAfterSatisfied(t *testing.T) {
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		1: {ID: 1, RepositoryID: 1, Status: string(statemachine.TaskStatusSucceeded)},
		2: {ID: 2, RepositoryID: 1, Status: string(statemachine.TaskStatusPending), RunAfter: 1},
	}}
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Status: string(statemachine.RepoStatusReady)},
	}}
	svc := NewTaskService(&config.Config{}, taskRepo, repoRepo, nil)
	o, _ := orchestrator.NewOrchestrator(1, &fakeExecutor{})
	defer o.Stop()
	svc.SetOrchestrator(o)

	if err := svc.Enqueue(2); err != nil {
		t.Fatalf("enqueue error: %v", err)
	}
	if taskRepo.tasks[2].Status != string(statemachine.TaskStatusQueued) {
		t.Fatalf("task status should be queued, got %s", taskRepo.tasks[2].Status)
	}
	if got := o.GetQueueStatus().QueueLength; got != 1 {
		t.Fatalf("queue length should be 1, got %d", got)
	}
}

func TestTaskServiceEnqueuePendingTasks(t *testing.T) {
	taskRepo := &mockTaskRepo{tasks: map[uint]*model.Task{
		1: {ID: 1, RepositoryID: 1, Status: string(statemachine.TaskStatusSucceeded)},
		2: {ID: 2, RepositoryID: 1, Status: string(statemachine.TaskStatusPending), RunAfter: 1},
		3: {ID: 3, RepositoryID: 1, Status: string(statemachine.TaskStatusPending)},
	}}
	repoRepo := &mockRepoRepo{repos: map[uint]*model.Repository{
		1: {ID: 1, Status: string(statemachine.RepoStatusReady)},
	}}
	svc := NewTaskService(&config.Config{}, taskRepo, repoRepo, nil)
	o, _ := orchestrator.NewOrchestrator(1, &fakeExecutor{})
	defer o.Stop()
	svc.SetOrchestrator(o)

	svc.enqueuePendingTasks(context.Background())

	if taskRepo.tasks[2].Status != string(statemachine.TaskStatusQueued) {
		t.Fatalf("task 2 status should be queued, got %s", taskRepo.tasks[2].Status)
	}
	if taskRepo.tasks[3].Status != string(statemachine.TaskStatusQueued) {
		t.Fatalf("task 3 status should be queued, got %s", taskRepo.tasks[3].Status)
	}
	if got := o.GetQueueStatus().QueueLength; got != 2 {
		t.Fatalf("queue length should be 2, got %d", got)
	}
}
