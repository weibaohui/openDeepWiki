package service

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

// ActivityScheduler 活跃度调度器
// 负责定期检查仓库的下次更新时间，并在到期时触发更新任务
type ActivityScheduler struct {
	cfg         *config.Config
	repoRepo    repository.RepoRepository
	taskEventBus *eventbus.TaskEventBus
	startOnce   sync.Once
	stopOnce    sync.Once
	stopChan    chan struct{}
}

// NewActivityScheduler 创建活跃度调度器
func NewActivityScheduler(cfg *config.Config, repoRepo repository.RepoRepository, taskEventBus *eventbus.TaskEventBus) *ActivityScheduler {
	return &ActivityScheduler{
		cfg:         cfg,
		repoRepo:    repoRepo,
		taskEventBus: taskEventBus,
		stopChan:    make(chan struct{}),
	}
}

// Start 启动调度器
func (s *ActivityScheduler) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		go s.run(ctx)
		klog.V(6).Info("活跃度调度器已启动")
	})
}

// Stop 停止调度器
func (s *ActivityScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
		klog.V(6).Info("活跃度调度器已停止")
	})
}

// run 运行调度器主循环
func (s *ActivityScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Activity.CheckInterval)
	defer ticker.Stop()

	klog.V(6).Infof("活跃度调度器运行中，检查间隔: %v", s.cfg.Activity.CheckInterval)

	for {
		select {
		case <-ctx.Done():
			klog.V(6).Info("活跃度调度器收到上下文取消信号")
			return
		case <-s.stopChan:
			klog.V(6).Info("活跃度调度器收到停止信号")
			return
		case <-ticker.C:
			s.checkAndTriggerUpdates(ctx)
		}
	}
}

// checkAndTriggerUpdates 检查并触发更新
func (s *ActivityScheduler) checkAndTriggerUpdates(ctx context.Context) {
	if !s.cfg.Activity.Enabled {
		return
	}

	now := time.Now()
	klog.V(6).Info("开始检查仓库更新时间...")

	// 获取所有仓库
	repos, err := s.repoRepo.List()
	if err != nil {
		klog.Errorf("获取仓库列表失败: %v", err)
		return
	}

	// 检查每个仓库
	for _, repo := range repos {
		if repo.NextUpdateTime == nil {
			continue
		}

		// 检查是否到达更新时间
		if now.After(*repo.NextUpdateTime) || now.Equal(*repo.NextUpdateTime) {
			klog.V(6).Infof("仓库到达更新时间: repoID=%d, nextUpdateTime=%v", repo.ID, *repo.NextUpdateTime)

			// 检查仓库状态，只有已完成的仓库才会触发更新
			currentStatus := statemachine.RepositoryStatus(repo.Status)
			if currentStatus != statemachine.RepoStatusCompleted {
				klog.V(6).Infof("仓库状态不为已完成，跳过更新: repoID=%d, status=%s", repo.ID, currentStatus)
				continue
			}

			// 触发增量分析任务
			s.triggerIncrementalAnalysis(ctx, repo.ID)
		}
	}

	klog.V(6).Info("仓库更新时间检查完成")
}

// triggerIncrementalAnalysis 触发增量分析任务
func (s *ActivityScheduler) triggerIncrementalAnalysis(ctx context.Context, repoID uint) {
	klog.V(6).Infof("为仓库触发增量分析: repoID=%d", repoID)

	// 发布增量分析事件到任务事件总线
	// 这里需要根据实际的增量分析事件结构进行调整
	// 暂时记录日志，后续可以添加具体的事件发布逻辑
	klog.V(6).Infof("仓库增量分析触发待实现: repoID=%d", repoID)
}

// UpdateNextUpdateTime 更新仓库的下一次更新时间
func (s *ActivityScheduler) UpdateNextUpdateTime(repoID uint, activityPoints int) error {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return err
	}

	now := time.Now()
	decreaseDuration := time.Duration(activityPoints) * s.cfg.Activity.DecreaseUnit
	newInterval := s.cfg.Activity.DefaultInterval - decreaseDuration

	// 确保最小间隔不小于1小时
	minInterval := 1 * time.Hour
	if newInterval < minInterval {
		newInterval = minInterval
	}

	newNextUpdateTime := now.Add(newInterval)
	repo.NextUpdateTime = &newNextUpdateTime

	if err := s.repoRepo.Save(repo); err != nil {
		return err
	}

	klog.V(6).Infof("更新仓库下次更新时间: repoID=%d, nextUpdateTime=%v", repoID, newNextUpdateTime)
	return nil
}
