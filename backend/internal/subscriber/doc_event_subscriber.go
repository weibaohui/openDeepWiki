package subscriber

import (
	"context"

	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type DocEventSubscriber struct {
	taskBus       *eventbus.TaskEventBus
	syncEventRepo repository.SyncEventRepository
}

func NewDocEventSubscriber(taskBus *eventbus.TaskEventBus, syncEventRepo repository.SyncEventRepository) *DocEventSubscriber {
	return &DocEventSubscriber{taskBus: taskBus, syncEventRepo: syncEventRepo}
}

func (s *DocEventSubscriber) Register(bus *eventbus.DocEventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(eventbus.DocEventRated, s.handleDocRated)
	bus.Subscribe(eventbus.DocEventPulled, s.handleDocPulled)
	bus.Subscribe(eventbus.DocEventPushed, s.handleDocPushed)
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

// handleDocPulled 处理文档被拉取事件
func (s *DocEventSubscriber) handleDocPulled(ctx context.Context, event eventbus.DocEvent) error {
	if s.syncEventRepo != nil {
		if err := s.syncEventRepo.Create(ctx, &model.SyncEvent{
			EventType:    string(event.Type),
			RepositoryID: event.RepositoryID,
			DocID:        event.DocID,
			TargetServer: event.TargetServer,
			Success:      event.Success,
		}); err != nil {
			klog.Errorf("文档拉取事件落库失败: type=%s, repositoryID=%d, docID=%d, target=%s, success=%t, error=%v", event.Type, event.RepositoryID, event.DocID, event.TargetServer, event.Success, err)
			return err
		}
	}
	klog.V(6).Infof("文档拉取事件处理成功: type=%s, repositoryID=%d, docID=%d, target=%s, success=%t", event.Type, event.RepositoryID, event.DocID, event.TargetServer, event.Success)
	return nil
}

// handleDocPushed 处理文档被推送事件
func (s *DocEventSubscriber) handleDocPushed(ctx context.Context, event eventbus.DocEvent) error {
	if s.syncEventRepo != nil {
		if err := s.syncEventRepo.Create(ctx, &model.SyncEvent{
			EventType:    string(event.Type),
			RepositoryID: event.RepositoryID,
			DocID:        event.DocID,
			TargetServer: event.TargetServer,
			Success:      event.Success,
		}); err != nil {
			klog.Errorf("文档推送事件落库失败: type=%s, repositoryID=%d, docID=%d, target=%s, success=%t, error=%v", event.Type, event.RepositoryID, event.DocID, event.TargetServer, event.Success, err)
			return err
		}
	}
	klog.V(6).Infof("文档推送事件处理成功: type=%s, repositoryID=%d, docID=%d, target=%s, success=%t", event.Type, event.RepositoryID, event.DocID, event.TargetServer, event.Success)
	return nil
}
