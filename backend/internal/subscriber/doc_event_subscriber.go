package subscriber

import (
	"context"

	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"k8s.io/klog/v2"
)

type DocEventSubscriber struct {
	taskBus *eventbus.TaskEventBus
}

func NewDocEventSubscriber(taskBus *eventbus.TaskEventBus) *DocEventSubscriber {
	return &DocEventSubscriber{taskBus: taskBus}
}

func (s *DocEventSubscriber) Register(bus *eventbus.DocEventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(eventbus.DocEventRated, s.handleDocRated)
}

func (s *DocEventSubscriber) handleDocRated(ctx context.Context, event eventbus.DocEvent) error {
	//todo 计算评分如果连续很低，那么触发文档重新生成
	// if event.Rating < 3 {
	// 	//todo 触发文档重新生成
	// 	s.taskBus.Publish(ctx, eventbus.TaskEventRegenerate, eventbus.TaskEvent{
	// 		Type:         eventbus.TaskEventRegenerate,
	// 		RepositoryID: event.RepositoryID,
	// 		DocID:        event.DocID,
	// 	})
	// }

	klog.V(6).Infof("文档事件处理成功: type=%s, repositoryID=%d, docID=%d, rating=%d", event.Type, event.RepositoryID, event.DocID, event.Rating)
	return nil
}
