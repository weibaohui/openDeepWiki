package subscriber

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"k8s.io/klog/v2"
)

type TaskEventSubscriber struct {
	taskService taskEventService
}

type taskEventService interface {
	CreateDocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int, writerNames ...domain.WriterName) (*model.Task, error)
	CreateTocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int) (*model.Task, error)
	CreateTitleRewriteTask(ctx context.Context, repoID uint, title string, runAfter uint, docId uint, sortOrder int) (*model.Task, error)
	CreateUserRequestTask(ctx context.Context, repoID uint, content string, sortOrder int) (*model.Task, error)
}

func NewTaskEventSubscriber(taskService taskEventService) *TaskEventSubscriber {
	return &TaskEventSubscriber{taskService: taskService}
}

func (s *TaskEventSubscriber) Register(bus *eventbus.TaskEventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(eventbus.TaskEventDocWrite, s.handleDocWrite)
	bus.Subscribe(eventbus.TaskEventTocWrite, s.handleTocWrite)
	bus.Subscribe(eventbus.TaskEventTitleRewrite, s.handleTitleRewrite)
	bus.Subscribe(eventbus.TaskEventUserRequest, s.handleUserRequest)
}

func (s *TaskEventSubscriber) handleDocWrite(ctx context.Context, event eventbus.TaskEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}
	var taskErr error
	var taskID uint
	if event.WriterName != "" {
		task, err := s.taskService.CreateDocWriteTask(ctx, event.RepositoryID, event.Title, event.SortOrder, event.WriterName)
		if err != nil {
			taskErr = err
		} else {
			taskID = task.ID
		}
	} else {
		task, err := s.taskService.CreateDocWriteTask(ctx, event.RepositoryID, event.Title, event.SortOrder)
		if err != nil {
			taskErr = err
		} else {
			taskID = task.ID
		}
	}
	if taskErr != nil {
		klog.Errorf("任务事件处理失败: type=%s, repoID=%d, error=%v", event.Type, event.RepositoryID, taskErr)
		return taskErr
	}
	klog.V(6).Infof("任务事件处理成功: type=%s, repoID=%d, taskID=%d", event.Type, event.RepositoryID, taskID)
	return nil
}

func (s *TaskEventSubscriber) handleTocWrite(ctx context.Context, event eventbus.TaskEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}
	task, err := s.taskService.CreateTocWriteTask(ctx, event.RepositoryID, event.Title, event.SortOrder)
	if err != nil {
		klog.Errorf("任务事件处理失败: type=%s, repoID=%d, error=%v", event.Type, event.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("任务事件处理成功: type=%s, repoID=%d, taskID=%d", event.Type, event.RepositoryID, task.ID)
	return nil
}

func (s *TaskEventSubscriber) handleTitleRewrite(ctx context.Context, event eventbus.TaskEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}
	if event.DocID == 0 {
		return fmt.Errorf("文档ID为空")
	}
	task, err := s.taskService.CreateTitleRewriteTask(ctx, event.RepositoryID, event.Title, event.RunAfter, event.DocID, event.SortOrder)
	if err != nil {
		klog.Errorf("任务事件处理失败: type=%s, repoID=%d, error=%v", event.Type, event.RepositoryID, err)
		return err
	}
	klog.V(6).Infof("任务事件处理成功: type=%s, repoID=%d, taskID=%d", event.Type, event.RepositoryID, task.ID)
	return nil
}

func (s *TaskEventSubscriber) handleUserRequest(ctx context.Context, event eventbus.TaskEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}
	task, err := s.taskService.CreateUserRequestTask(ctx, event.RepositoryID, event.Title, event.SortOrder)
	if err != nil {
		klog.Errorf("任务事件处理失败: type=%s, repoID=%d, error=%v", event.Type, event.RepositoryID, err)
		return err
	}

	klog.V(6).Infof("任务事件处理成功: type=%s, repoID=%d, taskID=%d", event.Type, event.RepositoryID, task.ID)
	return nil
}
