package eventbus

import (
	"context"

	"k8s.io/klog/v2"
)

// VectorEventHandler 向量事件处理器接口
type VectorEventHandler interface {
	GenerateForDocument(ctx context.Context, docID uint) error
	RegenerateForDocument(ctx context.Context, docID uint) error
}

// VectorEventSubscriber 向量事件订阅器
// 监听文档保存和更新事件，自动触发向量生成
type VectorEventSubscriber struct {
	handler VectorEventHandler
}

// NewVectorEventSubscriber 创建向量事件订阅器
func NewVectorEventSubscriber(handler VectorEventHandler) *VectorEventSubscriber {
	return &VectorEventSubscriber{
		handler: handler,
	}
}

// Subscribe 订阅文档事件
func (s *VectorEventSubscriber) Subscribe(bus *DocEventBus) {
	// 订阅文档保存事件
	bus.Subscribe(DocEventSaved, s.handleDocSaved)

	// 订阅文档更新事件
	bus.Subscribe(DocEventUpdated, s.handleDocUpdated)

	klog.V(6).Infof("VectorEventSubscriber: 已订阅文档事件")
}

// handleDocSaved 处理文档保存事件
func (s *VectorEventSubscriber) handleDocSaved(ctx context.Context, event DocEvent) error {
	klog.V(6).Infof("VectorEventSubscriber: 收到文档保存事件，docID: %d", event.DocID)

	// 异步生成向量，不阻塞主流程
	go func() {
		if err := s.handler.GenerateForDocument(ctx, event.DocID); err != nil {
			klog.Warningf("VectorEventSubscriber: 生成向量失败，docID: %d, error: %v", event.DocID, err)
		}
	}()

	return nil
}

// handleDocUpdated 处理文档更新事件
func (s *VectorEventSubscriber) handleDocUpdated(ctx context.Context, event DocEvent) error {
	klog.V(6).Infof("VectorEventSubscriber: 收到文档更新事件，docID: %d", event.DocID)

	// 异步重新生成向量
	go func() {
		if err := s.handler.RegenerateForDocument(ctx, event.DocID); err != nil {
			klog.Warningf("VectorEventSubscriber: 重新生成向量失败，docID: %d, error: %v", event.DocID, err)
		}
	}()

	return nil
}