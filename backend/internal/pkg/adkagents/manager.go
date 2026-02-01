package adkagents

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// Config Manager 配置
type Config struct {
	Dir            string
	AutoReload     bool
	ReloadInterval time.Duration

	// 依赖注入
	ModelProvider ModelProvider
	ToolProvider  ToolProvider
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Dir:            "./agents",
		AutoReload:     true,
		ReloadInterval: 5 * time.Second,
	}
}

// Manager ADK Agent 管理器
type Manager struct {
	config   *Config
	registry *Registry
	cache    map[string]adk.Agent // ADK Agent 实例缓存
	cacheMu  sync.RWMutex

	parser  *Parser
	loader  *Loader
	watcher *FileWatcher
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 检查依赖
	if config.ModelProvider == nil {
		return nil, fmt.Errorf("ModelProvider is required")
	}
	if config.ToolProvider == nil {
		return nil, fmt.Errorf("ToolProvider is required")
	}

	// 解析目录
	dir, err := resolveAgentsDir(config.Dir)
	if err != nil {
		return nil, err
	}
	config.Dir = dir

	log.Printf("[Manager] ADK Agents directory: %s", dir)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	// 创建组件
	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	m := &Manager{
		config:   config,
		registry: registry,
		cache:    make(map[string]adk.Agent),
		parser:   parser,
		loader:   loader,
	}

	// 初始加载
	results, err := loader.LoadFromDir(dir)
	if err != nil {
		log.Printf("[Manager] Warning: failed to load agents: %v", err)
	} else {
		loaded := 0
		failed := 0
		for _, r := range results {
			switch r.Action {
			case "created":
				loaded++
			case "failed":
				failed++
				if r.Agent != nil && r.Agent.Path != "" {
					log.Printf("[Manager] Failed to load agent from %s: %v", r.Agent.Path, r.Error)
				} else {
					log.Printf("[Manager] Failed to load agent: %v", r.Error)
				}
			}
		}
		if loaded > 0 {
			log.Printf("[Manager] Loaded %d agents", loaded)
		}
		if failed > 0 {
			log.Printf("[Manager] Failed to load %d agents", failed)
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
	m.watcher = NewFileWatcher(m.config.Dir, m.config.ReloadInterval, func(event FileEvent) {
		switch event.Type {
		case "create":
			log.Printf("[Manager] Loading new agent from %s", event.Path)
			if _, err := m.loader.LoadFromPath(event.Path); err != nil {
				log.Printf("[Manager] Failed to load agent: %v", err)
			} else {
				log.Printf("[Manager] Successfully loaded agent from %s", event.Path)
			}

		case "modify":
			agentName := guessAgentNameFromPath(event.Path)
			log.Printf("[Manager] Reloading agent: %s", agentName)
			if _, err := m.loader.Reload(agentName); err != nil {
				log.Printf("[Manager] Failed to reload agent: %v", err)
			} else {
				log.Printf("[Manager] Successfully reloaded agent: %s", agentName)
				// 清除缓存
				m.clearCache(agentName)
			}

		case "delete":
			agentName := guessAgentNameFromPath(event.Path)
			log.Printf("[Manager] Unloading agent: %s", agentName)
			if err := m.loader.Unload(agentName); err != nil {
				log.Printf("[Manager] Failed to unload agent: %v", err)
			} else {
				log.Printf("[Manager] Successfully unloaded agent: %s", agentName)
				// 清除缓存
				m.clearCache(agentName)
			}
		}
	})

	if err := m.watcher.Start(); err != nil {
		log.Printf("[Manager] Warning: failed to start file watcher: %v", err)
	}
}

// Stop 停止 Manager
func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
}

// GetAgent 获取指定名称的 ADK Agent 实例
// 如果缓存中不存在，根据 AgentDefinition 创建并缓存
func (m *Manager) GetAgent(name string) (adk.Agent, error) {
	// 先检查缓存
	m.cacheMu.RLock()
	if agent, exists := m.cache[name]; exists {
		m.cacheMu.RUnlock()
		return agent, nil
	}
	m.cacheMu.RUnlock()

	// 从注册表获取定义
	def, err := m.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// 创建 ADK Agent
	agent, err := m.createADKAgent(def)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	// 存入缓存
	m.cacheMu.Lock()
	m.cache[name] = agent
	m.cacheMu.Unlock()

	return agent, nil
}

// createADKAgent 根据 AgentDefinition 创建 ADK Agent
func (m *Manager) createADKAgent(def *AgentDefinition) (adk.Agent, error) {
	ctx := context.Background()

	// 获取模型
	chatModel, err := m.config.ModelProvider.GetModel(def.Model)
	if err != nil {
		log.Printf("[Manager] Failed to get model '%s', using default: %v", def.Model, err)
		chatModel = m.config.ModelProvider.DefaultModel()
	}

	// 获取工具
	tools := make([]tool.BaseTool, 0, len(def.Tools))
	for _, toolName := range def.Tools {
		t, err := m.config.ToolProvider.GetTool(toolName)
		if err != nil {
			log.Printf("[Manager] Warning: tool '%s' not found, skipping: %v", toolName, err)
			continue
		}
		tools = append(tools, t)
	}

	// 构造配置
	config := &adk.ChatModelAgentConfig{
		Name:         def.Name,
		Description:  def.Description,
		Instruction:  def.Instruction,
		Model:        chatModel,
		MaxIterations: def.MaxIterations,
	}

	// 如果有工具，添加 ToolsConfig
	if len(tools) > 0 {
		config.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		}
	}

	// 如果有退出配置，添加 Exit
	if def.Exit.Type != "" {
		config.Exit = adk.ExitTool{}
	}

	// 创建 Agent
	agent, err := adk.NewChatModelAgent(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChatModelAgent: %w", err)
	}

	return agent, nil
}

// List 列出所有 Agent 定义
func (m *Manager) List() []*AgentDefinition {
	return m.registry.List()
}

// Reload 重新加载指定 Agent
func (m *Manager) Reload(name string) error {
	_, err := m.loader.Reload(name)
	if err != nil {
		return err
	}
	m.clearCache(name)
	return nil
}

// clearCache 清除指定 Agent 的缓存
func (m *Manager) clearCache(name string) {
	m.cacheMu.Lock()
	delete(m.cache, name)
	m.cacheMu.Unlock()
}

// resolveAgentsDir 解析 Agents 目录
func resolveAgentsDir(configDir string) (string, error) {
	// 1. 环境变量
	if dir := os.Getenv("ADK_AGENTS_DIR"); dir != "" {
		return filepath.Abs(dir)
	}

	// 2. 配置
	if configDir != "" {
		return filepath.Abs(configDir)
	}

	// 3. 默认（当前工作目录）
	cwd, err := os.Getwd()
	if err != nil {
		return "./agents", nil
	}
	return filepath.Join(cwd, "agents"), nil
}

// guessAgentNameFromPath 从路径猜测 Agent name
func guessAgentNameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
