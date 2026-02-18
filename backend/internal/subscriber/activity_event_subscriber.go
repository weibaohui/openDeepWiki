package subscriber

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

// ActivityEventSubscriber 活跃度事件订阅者
type ActivityEventSubscriber struct {
	repoRepo repository.RepoRepository
	cfg      *config.ActivityConfig
}

// NewActivityEventSubscriber 创建活跃度事件订阅者
func NewActivityEventSubscriber(repoRepo repository.RepoRepository, cfg *config.Config) *ActivityEventSubscriber {
	return &ActivityEventSubscriber{
		repoRepo: repoRepo,
		cfg:      &cfg.Activity,
	}
}

// OnBrowseActivity 处理浏览活跃度事件
func (s *ActivityEventSubscriber) OnBrowseActivity(event eventbus.BrowseActivityEvent) {
	if !s.cfg.Enabled {
		return
	}

	// 获取仓库基本信息
	repo, err := s.repoRepo.GetBasic(event.RepositoryID)
	if err != nil {
		klog.V(6).Infof("获取仓库基本信息失败: repoID=%d, error=%v", event.RepositoryID, err)
		return
	}

	now := time.Now()

	// 检查是否需要重置今日活跃度
	needReset := false
	if repo.LastActivityResetDate == nil {
		needReset = true
	} else {
		// 如果日期不同，需要重置
		lastReset := repo.LastActivityResetDate.In(now.Location())
		if lastReset.Year() != now.Year() || lastReset.Month() != now.Month() || lastReset.Day() != now.Day() {
			needReset = true
		}
	}

	if needReset {
		// 重置今日活跃度
		repo.TodayActivityCount = 0
		repo.LastActivityResetDate = &now
		klog.V(6).Infof("重置仓库今日活跃度: repoID=%d", event.RepositoryID)
	}

	// 增加活跃度点数
	repo.TodayActivityCount++

	// 调整下次更新时间
	// 计算新的更新时间间隔：默认间隔 - 活跃点数 * 单位调减时间
	decreaseDuration := time.Duration(repo.TodayActivityCount) * s.cfg.DecreaseUnit
	newInterval := s.cfg.DefaultInterval - decreaseDuration

	// 确保最小间隔不小于1小时
	minInterval := 1 * time.Hour
	if newInterval < minInterval {
		newInterval = minInterval
	}

	// 计算下次更新时间
	newNextUpdateTime := now.Add(newInterval)
	repo.NextUpdateTime = &newNextUpdateTime

	// 保存更新
	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库活跃度信息失败: repoID=%d, error=%v", event.RepositoryID, err)
		return
	}

	klog.V(6).Infof("更新仓库活跃度成功: repoID=%d, activityCount=%d, nextUpdateTime=%v",
		event.RepositoryID, repo.TodayActivityCount, newNextUpdateTime)
}

// Register 注册订阅者到事件总线
func (s *ActivityEventSubscriber) Register(bus *eventbus.ActivityEventBus) {
	subscriber := eventbus.NewActivityEventSubscriber(s.OnBrowseActivity)
	subscriber.Register(bus)
}
