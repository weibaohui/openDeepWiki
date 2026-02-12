package subscriber

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"k8s.io/klog/v2"
)

type RepositoryEventSubscriber struct {
	taskService taskEventService
	repoService repositoryEventService
}

type repositoryEventService interface {
	CloneRepository(ctx context.Context, repoID uint) error
}

func NewRepositoryEventSubscriber(taskService taskEventService, repoService repositoryEventService) *RepositoryEventSubscriber {
	return &RepositoryEventSubscriber{taskService: taskService, repoService: repoService}
}

func (s *RepositoryEventSubscriber) Register(bus *eventbus.RepositoryEventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(eventbus.RepositoryEventAdded, s.handleRepoAdded)

}

func (s *RepositoryEventSubscriber) handleRepoAdded(ctx context.Context, event eventbus.RepositoryEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}

	// 异步克隆仓库
	if err := s.repoService.CloneRepository(ctx, event.RepositoryID); err != nil {
		klog.Errorf("CloneRepository failed: %v", err)
	}

	// 异步创建目录分析任务
	_, err := s.taskService.CreateTocWriteTask(ctx, event.RepositoryID, "目录分析", 10)
	if err != nil {
		klog.Errorf("CreateTocWriteTask failed: %v", err)
	}

	klog.V(6).Infof("仓库事件处理成功: type=%s, repoID=%d", event.Type, event.RepositoryID)
	return nil
}
