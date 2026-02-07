package repository

import (
	"errors"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// ErrNotFound 记录不存在错误
var ErrNotFound = errors.New("record not found")

type RepoRepository interface {
	Create(repo *model.Repository) error
	List() ([]model.Repository, error)
	Get(id uint) (*model.Repository, error)
	GetBasic(id uint) (*model.Repository, error)
	Save(repo *model.Repository) error
	Delete(id uint) error
}

type TaskRepository interface {
	Create(task *model.Task) error
	GetByRepository(repoID uint) ([]model.Task, error)
	GetByStatus(status string) ([]model.Task, error)
	Get(id uint) (*model.Task, error)
	Save(task *model.Task) error
	CleanupStuckTasks(timeout time.Duration) (int64, error)
	GetStuckTasks(timeout time.Duration) ([]model.Task, error)
	DeleteByRepositoryID(repoID uint) error
	Delete(id uint) error
	GetTaskStats(repoID uint) (map[string]int64, error)
	GetActiveTasks() ([]model.Task, error)
	GetRecentTasks(limit int) ([]model.Task, error)
}

type DocumentRepository interface {
	Create(doc *model.Document) error
	GetByRepository(repoID uint) ([]model.Document, error)
	Get(id uint) (*model.Document, error)
	Save(doc *model.Document) error
	Delete(id uint) error
	DeleteByTaskID(taskID uint) error
	DeleteByRepositoryID(repoID uint) error

	CreateVersioned(doc *model.Document) error
	GetLatestVersionByTaskID(taskID uint) (int, error)
	ClearLatestByTaskID(taskID uint) error
	GetByTaskID(taskID uint) ([]model.Document, error)
}

type EvidenceRepository interface {
	CreateBatch(evidences []model.TaskEvidence) error
	GetByTaskID(taskID uint) ([]model.TaskEvidence, error)
}
