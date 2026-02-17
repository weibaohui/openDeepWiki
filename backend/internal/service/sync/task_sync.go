package syncservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gorm.io/gorm"
)

// TaskSyncService 任务同步服务
type TaskSyncService struct {
	repoRepo repository.RepoRepository
	taskRepo repository.TaskRepository
}

// NewTaskSyncService 创建新的任务同步服务
func NewTaskSyncService(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository) *TaskSyncService {
	return &TaskSyncService{
		repoRepo: repoRepo,
		taskRepo: taskRepo,
	}
}

// Create 创建任务
func (s *TaskSyncService) Create(ctx context.Context, req syncdto.TaskCreateRequest) (*model.Task, error) {
	repo, err := s.repoRepo.GetBasic(req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	writerName := domain.WriterName(req.WriterName)
	if writerName == "" {
		writerName = domain.DefaultWriter
	}
	taskType := domain.TaskType(req.TaskType)
	if taskType == "" {
		taskType = domain.DocWrite
	}

	task := &model.Task{
		ID:           req.TaskID,
		RepositoryID: repo.ID,
		DocID:        req.DocID,
		WriterName:   writerName,
		TaskType:     taskType,
		Title:        req.Title,
		Outline:      req.Outline,
		Status:       req.Status,
		RunAfter:     req.RunAfter,
		ErrorMsg:     req.ErrorMsg,
		SortOrder:    req.SortOrder,
		StartedAt:    req.StartedAt,
		CompletedAt:  req.CompletedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if req.TaskID != 0 {
		existing, err := s.taskRepo.Get(req.TaskID)
		if err == nil {
			existing.RepositoryID = task.RepositoryID
			existing.DocID = task.DocID
			existing.WriterName = task.WriterName
			existing.TaskType = task.TaskType
			existing.Title = task.Title
			existing.Outline = task.Outline
			existing.Status = task.Status
			existing.RunAfter = task.RunAfter
			existing.ErrorMsg = task.ErrorMsg
			existing.SortOrder = task.SortOrder
			existing.StartedAt = task.StartedAt
			existing.CompletedAt = task.CompletedAt
			existing.CreatedAt = task.CreatedAt
			existing.UpdatedAt = task.UpdatedAt
			if err := s.taskRepo.Save(existing); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, domain.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateDocID 更新任务的文档ID
func (s *TaskSyncService) UpdateDocID(ctx context.Context, taskID uint, docID uint) (*model.Task, error) {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}
	task.DocID = docID
	task.UpdatedAt = time.Now()
	if err := s.taskRepo.Save(task); err != nil {
		return nil, err
	}
	return task, nil
}

// ToTaskData 将 model.Task 转换为 TaskData
func ToTaskData(task model.Task) TaskData {
	return TaskData{
		ID:           task.ID,
		RepositoryID: task.RepositoryID,
		DocID:        task.DocID,
		WriterName:   string(task.WriterName),
		TaskType:     string(task.TaskType),
		Title:        task.Title,
		Outline:      task.Outline,
		Status:       task.Status,
		RunAfter:     task.RunAfter,
		ErrorMsg:     task.ErrorMsg,
		SortOrder:    task.SortOrder,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}
