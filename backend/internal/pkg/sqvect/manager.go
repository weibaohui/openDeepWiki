package sqvect

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/klog/v2"
)

// Manager sqvect 管理器，负责管理向量存储实例
type Manager struct {
	mu     sync.RWMutex
	stores map[string]*Store // 按集合名称管理多个存储实例
	config Config
}

// NewManager 创建 sqvect 管理器
func NewManager(config Config) (*Manager, error) {
	m := &Manager{
		config: config,
		stores: make(map[string]*Store),
	}

	// 初始化默认存储
	store, err := NewStore(config)
	if err != nil {
		return nil, fmt.Errorf("create default store: %w", err)
	}
	m.stores[config.CollectionName] = store

	klog.V(4).Infof("sqvect manager initialized with collection: %s", config.CollectionName)
	return m, nil
}

// GetStore 获取默认存储实例
func (m *Manager) GetStore() *Store {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stores[m.config.CollectionName]
}

// GetStoreByCollection 根据集合名称获取存储实例
func (m *Manager) GetStoreByCollection(collection string) (*Store, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	store, ok := m.stores[collection]
	return store, ok
}

// CreateCollection 创建新集合
func (m *Manager) CreateCollection(ctx context.Context, name string, dimensions int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.stores[name]; ok {
		return nil // 集合已存在
	}

	config := m.config
	config.CollectionName = name
	config.Dimensions = dimensions

	store, err := NewStore(config)
	if err != nil {
		return fmt.Errorf("create store for collection %s: %w", name, err)
	}

	m.stores[name] = store
	klog.V(4).Infof("Created collection: %s", name)
	return nil
}

// Close 关闭所有存储实例
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, store := range m.stores {
		if err := store.Close(); err != nil {
			lastErr = err
			klog.Warningf("Failed to close store %s: %v", name, err)
		}
	}

	m.stores = make(map[string]*Store)
	return lastErr
}

// GetStats 获取所有存储的统计信息
func (m *Manager) GetStats(ctx context.Context) (map[string]*StoreStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]*StoreStats)
	for name, store := range m.stores {
		s, err := store.Stats(ctx)
		if err != nil {
			klog.Warningf("Failed to get stats for collection %s: %v", name, err)
			continue
		}
		stats[name] = s
	}

	return stats, nil
}

// DefaultManager 全局默认管理器
var (
	defaultManager     *Manager
	defaultManagerOnce sync.Once
	defaultManagerErr  error
)

// InitDefaultManager 初始化全局默认管理器
func InitDefaultManager(config Config) error {
	defaultManagerOnce.Do(func() {
		defaultManager, defaultManagerErr = NewManager(config)
	})
	return defaultManagerErr
}

// GetDefaultManager 获取全局默认管理器
func GetDefaultManager() (*Manager, error) {
	if defaultManager == nil {
		return nil, fmt.Errorf("sqvect manager not initialized, call InitDefaultManager first")
	}
	return defaultManager, nil
}

// GetDefaultStore 获取全局默认存储
func GetDefaultStore() (*Store, error) {
	m, err := GetDefaultManager()
	if err != nil {
		return nil, err
	}
	return m.GetStore(), nil
}
