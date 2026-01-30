package skills

import "sync"

// Provider Skill 提供者接口
type Provider interface {
	// Type 返回 Provider 类型标识
	Type() string

	// Create 根据配置创建 Skill 实例
	Create(config SkillConfig) (Skill, error)
}

// ProviderRegistry Provider 注册中心
type ProviderRegistry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewProviderRegistry 创建 Provider 注册中心
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register 注册 Provider
func (pr *ProviderRegistry) Register(provider Provider) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.providers[provider.Type()] = provider
}

// Get 获取 Provider
func (pr *ProviderRegistry) Get(providerType string) (Provider, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	provider, exists := pr.providers[providerType]
	if !exists {
		return nil, ErrProviderNotFound
	}

	return provider, nil
}

// List 列出所有已注册的 Providers
func (pr *ProviderRegistry) List() []Provider {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := make([]Provider, 0, len(pr.providers))
	for _, provider := range pr.providers {
		result = append(result, provider)
	}

	return result
}
