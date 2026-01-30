package agents

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config Manager 配置
type Config struct {
	Dir            string
	AutoReload     bool
	ReloadInterval time.Duration
	DefaultAgent   string
	Routes         map[string]string // entryPoint -> agentName
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Dir:            "./agents",
		AutoReload:     true,
		ReloadInterval: 5 * time.Second,
		Routes:         make(map[string]string),
	}
}

// Manager Agent 管理器
type Manager struct {
	Config   *Config
	Registry Registry
	Parser   *Parser
	Loader   *Loader
	Router   Router
	watcher  *FileWatcher
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 解析目录
	dir, err := resolveAgentsDir(config.Dir)
	if err != nil {
		return nil, err
	}
	config.Dir = dir

	log.Printf("Agents directory: %s", dir)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	// 创建组件
	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)
	router := NewRouter(registry)

	// 注册路由规则
	for entryPoint, agentName := range config.Routes {
		router.RegisterRoute(entryPoint, agentName)
	}

	m := &Manager{
		Config:   config,
		Registry: registry,
		Parser:   parser,
		Loader:   loader,
		Router:   router,
	}

	// 初始加载
	results, err := loader.LoadFromDir(dir)
	if err != nil {
		log.Printf("Warning: failed to load agents: %v", err)
	} else {
		loaded := 0
		updated := 0
		failed := 0
		for _, r := range results {
			switch r.Action {
			case "created":
				loaded++
			case "updated":
				updated++
			case "failed":
				failed++
				if r.Agent != nil && r.Agent.Path != "" {
					log.Printf("Failed to load agent from %s: %v", r.Agent.Path, r.Error)
				} else {
					log.Printf("Failed to load agent: %v", r.Error)
				}
			}
		}
		if loaded > 0 || updated > 0 {
			log.Printf("Loaded %d agents, updated %d agents", loaded, updated)
		}
		if failed > 0 {
			log.Printf("Failed to load %d agents", failed)
		}
	}

	// 设置默认 Agent
	if config.DefaultAgent != "" {
		if err := router.SetDefault(config.DefaultAgent); err != nil {
			log.Printf("Warning: failed to set default agent: %v", err)
		}
	}

	// 启动热加载
	if config.AutoReload {
		m.startWatcher()
	}

	return m, nil
}

// startWatcher 启动文件监听
func (m *Manager) startWatcher() {
	m.watcher = NewFileWatcher(m.Config.Dir, m.Config.ReloadInterval, func(event FileEvent) {
		switch event.Type {
		case "create":
			log.Printf("Loading new agent from %s", event.Path)
			if _, err := m.Loader.LoadFromPath(event.Path); err != nil {
				log.Printf("Failed to load agent: %v", err)
			} else {
				log.Printf("Successfully loaded agent from %s", event.Path)
			}

		case "modify":
			agentName := guessAgentNameFromPath(event.Path)
			// 获取当前 agent 的路径，确保我们重新加载的是同一个
			existingAgent, err := m.Registry.Get(agentName)
			if err != nil {
				// Agent 不存在，可能是新文件
				log.Printf("Loading agent from modified file: %s", event.Path)
				if _, err := m.Loader.LoadFromPath(event.Path); err != nil {
					log.Printf("Failed to load agent: %v", err)
				}
				return
			}

			log.Printf("Reloading agent: %s", agentName)
			if _, err := m.Loader.Reload(agentName); err != nil {
				log.Printf("Failed to reload agent: %v", err)
			} else {
				log.Printf("Successfully reloaded agent: %s", agentName)
				// 如果路径发生变化（文件名改了但内容里的 name 没变），更新路径
				if existingAgent.Path != event.Path {
					existingAgent.Path = event.Path
				}
			}

		case "delete":
			agentName := guessAgentNameFromPath(event.Path)
			// 检查该 agent 是否确实使用这个路径
			agent, err := m.Registry.Get(agentName)
			if err != nil {
				return // Agent 已不存在
			}
			// 只有当路径匹配时才卸载（避免误删其他同名 agent）
			if agent.Path == event.Path {
				log.Printf("Unloading agent: %s", agentName)
				if err := m.Loader.Unload(agentName); err != nil {
					log.Printf("Failed to unload agent: %v", err)
				} else {
					log.Printf("Successfully unloaded agent: %s", agentName)
				}
			}
		}
	})

	if err := m.watcher.Start(); err != nil {
		log.Printf("Warning: failed to start file watcher: %v", err)
	}
}

// Stop 停止 Manager
func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
}

// SelectAgent 根据上下文选择 Agent
func (m *Manager) SelectAgent(ctx RouterContext) (*Agent, error) {
	return m.Router.Route(ctx)
}

// ReloadAll 重新加载所有 Agents
func (m *Manager) ReloadAll() error {
	// 获取当前所有 agents
	agents := m.Registry.List()

	// 卸载所有
	for _, agent := range agents {
		if err := m.Loader.Unload(agent.Name); err != nil {
			log.Printf("Failed to unload agent %s: %v", agent.Name, err)
		}
	}

	// 重新加载
	_, err := m.Loader.LoadFromDir(m.Config.Dir)
	return err
}

// resolveAgentsDir 解析 Agents 目录
func resolveAgentsDir(configDir string) (string, error) {
	// 1. 环境变量
	if dir := os.Getenv("AGENTS_DIR"); dir != "" {
		return filepath.Abs(dir)
	}

	// 2. 配置
	if configDir != "" {
		return filepath.Abs(configDir)
	}

	// 3. 默认
	exePath, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "agents"), nil
	}
	return filepath.Join(filepath.Dir(exePath), "agents"), nil
}

// guessAgentNameFromPath 从路径猜测 Agent name
func guessAgentNameFromPath(path string) string {
	// 从文件名提取（去掉扩展名）
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
