package eventbus

// BrowseActivityEvent 浏览活跃度事件
type BrowseActivityEvent struct {
	RepositoryID uint
	UserAgent    string
	Timestamp    int64
}

// ActivityEventBus 活跃度事件总线
type ActivityEventBus struct {
	publishers  []*ActivityEventPublisher
	subscribers []*ActivityEventSubscriber
}

// NewActivityEventBus 创建活跃度事件总线
func NewActivityEventBus() *ActivityEventBus {
	return &ActivityEventBus{
		publishers:  make([]*ActivityEventPublisher, 0),
		subscribers: make([]*ActivityEventSubscriber, 0),
	}
}

// Publish 发布浏览活跃度事件
func (b *ActivityEventBus) Publish(event BrowseActivityEvent) {
	for _, sub := range b.subscribers {
		sub.OnBrowseActivity(event)
	}
}

// ActivityEventPublisher 活跃度事件发布者
type ActivityEventPublisher struct {
	bus *ActivityEventBus
}

// NewActivityEventPublisher 创建活跃度事件发布者
func NewActivityEventPublisher(bus *ActivityEventBus) *ActivityEventPublisher {
	publisher := &ActivityEventPublisher{bus: bus}
	bus.publishers = append(bus.publishers, publisher)
	return publisher
}

// PublishBrowseActivity 发布浏览活跃度事件
func (p *ActivityEventPublisher) PublishBrowseActivity(repoID uint, userAgent string) {
	p.bus.Publish(BrowseActivityEvent{
		RepositoryID: repoID,
		UserAgent:    userAgent,
		Timestamp:    0, // 由订阅者设置
	})
}

// ActivityEventSubscriber 活跃度事件订阅者
type ActivityEventSubscriber struct {
	onBrowseActivity func(BrowseActivityEvent)
}

// NewActivityEventSubscriber 创建活跃度事件订阅者
func NewActivityEventSubscriber(onBrowseActivity func(BrowseActivityEvent)) *ActivityEventSubscriber {
	return &ActivityEventSubscriber{
		onBrowseActivity: onBrowseActivity,
	}
}

// OnBrowseActivity 处理浏览活跃度事件
func (s *ActivityEventSubscriber) OnBrowseActivity(event BrowseActivityEvent) {
	if s.onBrowseActivity != nil {
		s.onBrowseActivity(event)
	}
}

// Register 注册订阅者到事件总线
func (s *ActivityEventSubscriber) Register(bus *ActivityEventBus) {
	bus.subscribers = append(bus.subscribers, s)
}
