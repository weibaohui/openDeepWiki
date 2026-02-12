package eventsubscriber

import (
	"context"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockTaskEventService struct {
	docWriteCalled     int
	tocWriteCalled     int
	titleRewriteCalled int
	userRequestCalled  int
}

func (m *mockTaskEventService) CreateDocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int, writerNames ...domain.WriterName) (*model.Task, error) {
	m.docWriteCalled++
	return &model.Task{ID: 1}, nil
}

func (m *mockTaskEventService) CreateTocWriteTask(ctx context.Context, repoID uint, title string, sortOrder int) (*model.Task, error) {
	m.tocWriteCalled++
	return &model.Task{ID: 2}, nil
}

func (m *mockTaskEventService) CreateTitleRewriteTask(ctx context.Context, repoID uint, title string, runAfter uint, docId uint, sortOrder int) (*model.Task, error) {
	m.titleRewriteCalled++
	return &model.Task{ID: 3}, nil
}

func (m *mockTaskEventService) CreateUserRequestTask(ctx context.Context, repoID uint, content string, sortOrder int) (*model.Task, error) {
	m.userRequestCalled++
	return &model.Task{ID: 4}, nil
}

func TestTaskEventSubscriberRegisterAndHandle(t *testing.T) {
	bus := eventbus.NewBus()
	mockSvc := &mockTaskEventService{}
	subscriber := NewTaskEventSubscriber(mockSvc)
	subscriber.Register(bus)

	if err := bus.Publish(context.Background(), eventbus.TaskEvent{Type: eventbus.TaskEventDocWrite, RepositoryID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bus.Publish(context.Background(), eventbus.TaskEvent{Type: eventbus.TaskEventTocWrite, RepositoryID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bus.Publish(context.Background(), eventbus.TaskEvent{Type: eventbus.TaskEventTitleRewrite, RepositoryID: 1, DocID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bus.Publish(context.Background(), eventbus.TaskEvent{Type: eventbus.TaskEventUserRequest, RepositoryID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockSvc.docWriteCalled != 1 || mockSvc.tocWriteCalled != 1 || mockSvc.titleRewriteCalled != 1 || mockSvc.userRequestCalled != 1 {
		t.Fatalf("unexpected call counts: %+v", mockSvc)
	}
}
