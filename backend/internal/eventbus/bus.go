package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"k8s.io/klog/v2"
)

type Handler[E any] func(ctx context.Context, event E) error

type Bus[K comparable, E any] struct {
	mutex       sync.RWMutex
	subscribers map[K]map[uint64]Handler[E]
	counter     uint64
}

func NewBus[K comparable, E any]() *Bus[K, E] {
	return &Bus[K, E]{
		subscribers: make(map[K]map[uint64]Handler[E]),
	}
}

func (b *Bus[K, E]) Subscribe(eventType K, handler Handler[E]) func() {
	if handler == nil {
		return func() {}
	}
	id := atomic.AddUint64(&b.counter, 1)
	b.mutex.Lock()
	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = make(map[uint64]Handler[E])
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

func (b *Bus[K, E]) Publish(ctx context.Context, eventType K, event E) error {
	klog.V(6).Infof("广播 事件: type=%v, event=%v", eventType, event)
	b.mutex.RLock()
	handlersMap := b.subscribers[eventType]
	handlers := make([]Handler[E], 0, len(handlersMap))
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
