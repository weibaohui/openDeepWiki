package skills

import (
	"fmt"
	"log"
	"time"
)

// Manager Skills 管理器
type Manager struct {
	Registry  Registry
	Loader    *ConfigLoader
	Watcher   *FileWatcher
	providers *ProviderRegistry
}

// ManagerConfig Manager 配置
type ManagerConfig struct {
	// SkillsDir Skills 配置目录，如果为空则使用环境变量或默认目录
	SkillsDir string
	
	// Providers 自定义 Provider 列表，如果不指定则使用默认 Provider
	Providers []Provider
}

// NewManager 创建 Skills 管理器
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config == nil {
		config = &ManagerConfig{}
	}
	
	// 解析目录
	skillsDir, err := ResolveSkillsDir(config.SkillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve skills directory: %w", err)
	}

	log.Printf("Skills directory: %s", skillsDir)

	// 确保目录存在（如果不存在则创建）
	if err := EnsureDir(skillsDir); err != nil {
		return nil, fmt.Errorf("failed to ensure skills directory: %w", err)
	}

	// 创建 Registry
	registry := NewRegistry()

	// 创建 Provider 注册中心
	providers := NewProviderRegistry()

	// 注册自定义 Providers
	for _, provider := range config.Providers {
		providers.Register(provider)
		log.Printf("Registered provider: %s", provider.Type())
	}

	// 创建配置加载器
	loader := NewConfigLoader(registry, providers)

	// 初始加载
	if err := loader.LoadFromDir(skillsDir); err != nil {
		log.Printf("Warning: failed to load skills from %s: %v", skillsDir, err)
		// 不返回错误，允许空目录启动
	}

	// 创建文件监听器
	watcher := NewFileWatcher(skillsDir, 5*time.Second, func(event FileEvent) {
		switch event.Type {
		case "create", "modify":
			log.Printf("Loading skill from %s", event.Path)
			if err := loader.LoadFromFile(event.Path); err != nil {
				log.Printf("Failed to reload skill from %s: %v", event.Path, err)
			} else {
				log.Printf("Successfully loaded skill from %s", event.Path)
			}
		case "delete":
			log.Printf("Unloading skill from %s", event.Path)
			if err := loader.UnloadFromFile(event.Path); err != nil {
				log.Printf("Failed to unload skill from %s: %v", event.Path, err)
			} else {
				log.Printf("Successfully unloaded skill from %s", event.Path)
			}
		}
	})

	// 启动监听
	if err := watcher.Start(); err != nil {
		log.Printf("Warning: failed to start file watcher: %v", err)
		// 不返回错误，允许无文件监听运行
	}

	return &Manager{
		Registry:  registry,
		Loader:    loader,
		Watcher:   watcher,
		providers: providers,
	}, nil
}

// Stop 停止管理器
func (m *Manager) Stop() {
	if m.Watcher != nil {
		m.Watcher.Stop()
	}
}

// RegisterProvider 注册 Provider
func (m *Manager) RegisterProvider(provider Provider) {
	m.providers.Register(provider)
}

// GetProvider 获取 Provider
func (m *Manager) GetProvider(providerType string) (Provider, error) {
	return m.providers.Get(providerType)
}
