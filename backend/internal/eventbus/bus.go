package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Bus struct {
	mutex       sync.RWMutex
	subscribers map[TaskEventType]map[uint64]TaskEventHandler
	counter     uint64
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[TaskEventType]map[uint64]TaskEventHandler),
	}
}

func (b *Bus) Subscribe(eventType TaskEventType, handler TaskEventHandler) func() {
	if handler == nil {
		return func() {}
	}
	id := atomic.AddUint64(&b.counter, 1)
	b.mutex.Lock()
	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = make(map[uint64]TaskEventHandler)
	}
	b.subscribers[eventType][id] = handler
	b.mutex.Unlock()
	return func() {
		b.mutex.Lock()
		handlers, ok := b.subscribers[eventType]
		if ok {
			delete(handlers, id)
			if len(handlers) == 0 {
				delete(b.subscribers, eventType)
			}
		}
		b.mutex.Unlock()
	}
}

func (b *Bus) Publish(ctx context.Context, event TaskEvent) error {
	b.mutex.RLock()
	handlersMap := b.subscribers[event.Type]
	handlers := make([]TaskEventHandler, 0, len(handlersMap))
	for _, handler := range handlersMap {
		handlers = append(handlers, handler)
	}
	b.mutex.RUnlock()

	var errs []error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
