package subscriber

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"k8s.io/klog/v2"
)

type RepositoryEventSubscriber struct {
	taskBus     *eventbus.TaskEventBus
	taskService taskEventService
	repoService repositoryEventService
}

type repositoryEventService interface {
	CloneRepository(ctx context.Context, repoID uint) error
	UpdateRepositoryCloneInfo(ctx context.Context, repoID uint, branch string, commit string) error
}

func NewRepositoryEventSubscriber(taskBus *eventbus.TaskEventBus, taskService taskEventService, repoService repositoryEventService) *RepositoryEventSubscriber {
	return &RepositoryEventSubscriber{taskBus: taskBus, taskService: taskService, repoService: repoService}
}

func (s *RepositoryEventSubscriber) Register(bus *eventbus.RepositoryEventBus) {
	if bus == nil {
		return
	}
	bus.Subscribe(eventbus.RepositoryEventAdded, s.handleRepoAdded)
	bus.Subscribe(eventbus.RepositoryEventIncrementalUpdated, s.handleIncrementalUpdated)

}

func (s *RepositoryEventSubscriber) handleRepoAdded(ctx context.Context, event eventbus.RepositoryEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}

	// 异步克隆仓库
	if err := s.repoService.CloneRepository(ctx, event.RepositoryID); err != nil {
		klog.Errorf("CloneRepository failed: %v", err)
		return err
	}

	s.taskBus.Publish(ctx, eventbus.TaskEventTocWrite, eventbus.TaskEvent{
		Type:         eventbus.TaskEventTocWrite,
		RepositoryID: event.RepositoryID,
		Title:        "目录分析",
		SortOrder:    10,
		WriterName:   domain.TocWriter,
	})

	klog.V(6).Infof("仓库事件处理成功: type=%s, repoID=%d", event.Type, event.RepositoryID)
	return nil
}

// handleIncrementalUpdated 处理增量更新完成事件并更新仓库提交信息。
func (s *RepositoryEventSubscriber) handleIncrementalUpdated(ctx context.Context, event eventbus.RepositoryEvent) error {
	if event.RepositoryID == 0 {
		return fmt.Errorf("仓库ID为空")
	}
	if event.CloneCommit == "" {
		return fmt.Errorf("仓库最新提交为空")
	}
	if err := s.repoService.UpdateRepositoryCloneInfo(ctx, event.RepositoryID, event.CloneBranch, event.CloneCommit); err != nil {
		klog.V(6).Infof("增量更新提交信息落库失败: repoID=%d, branch=%s, commit=%s, error=%v", event.RepositoryID, event.CloneBranch, event.CloneCommit, err)
		return err
	}
	klog.V(6).Infof("增量更新提交信息落库成功: repoID=%d, branch=%s, commit=%s", event.RepositoryID, event.CloneBranch, event.CloneCommit)
	return nil
}
