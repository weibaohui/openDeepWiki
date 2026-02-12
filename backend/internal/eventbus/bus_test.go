package eventbus

import (
	"context"
	"errors"
	"testing"
)

func TestBusPublishBroadcast(t *testing.T) {
	bus := NewBus()
	calledA := false
	calledB := false

	bus.Subscribe(TaskEventDocWrite, func(ctx context.Context, event TaskEvent) error {
		calledA = true
		return nil
	})
	bus.Subscribe(TaskEventDocWrite, func(ctx context.Context, event TaskEvent) error {
		calledB = true
		return nil
	})

	if err := bus.Publish(context.Background(), TaskEvent{Type: TaskEventDocWrite}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledA || !calledB {
		t.Fatalf("expected handlers to be called")
	}
}

func TestBusUnsubscribe(t *testing.T) {
	bus := NewBus()
	called := false
	unsubscribe := bus.Subscribe(TaskEventDocWrite, func(ctx context.Context, event TaskEvent) error {
		called = true
		return nil
	})
	unsubscribe()

	if err := bus.Publish(context.Background(), TaskEvent{Type: TaskEventDocWrite}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatalf("expected handler to be unsubscribed")
	}
}

func TestBusPublishJoinErrors(t *testing.T) {
	bus := NewBus()
	bus.Subscribe(TaskEventDocWrite, func(ctx context.Context, event TaskEvent) error {
		return errors.New("err-a")
	})
	bus.Subscribe(TaskEventDocWrite, func(ctx context.Context, event TaskEvent) error {
		return errors.New("err-b")
	})

	if err := bus.Publish(context.Background(), TaskEvent{Type: TaskEventDocWrite}); err == nil {
		t.Fatalf("expected error")
	}
}
