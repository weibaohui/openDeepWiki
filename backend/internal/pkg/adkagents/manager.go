package adkagents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

// Manager ADK Agent 管理器
type Manager struct {
	cfg      *config.Config
	registry *Registry
	cache    map[string]adk.Agent // ADK Agent 实例缓存
	cacheMu  sync.RWMutex

	parser  *Parser
	loader  *Loader
	watcher *FileWatcher
}

var (
	managerInstance     *Manager
	managerInstanceOnce sync.Once
)

// GetOrCreateInstance 获取或创建 Manager 单例
func GetOrCreateInstance(cfg *config.Config) (*Manager, error) {
	managerInstanceOnce.Do(func() {
		instance, err := newManagerInternal(cfg)
		if err != nil {
			klog.Fatalf("[Manager] Failed to create manager: %v", err)
		}
		managerInstance = instance
	})

	return managerInstance, nil
}

// newManagerInternal 创建 Manager 实例（内部构造）
func newManagerInternal(cfg *config.Config) (*Manager, error) {

	// 创建组件
	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	m := &Manager{
		cfg:      cfg,
		registry: registry,
		cache:    make(map[string]adk.Agent),
		parser:   parser,
		loader:   loader,
	}

	// 初始加载
	_, _ = loader.LoadFromDir(cfg.Agent.Dir)

	// 启动热加载
	m.startWatcher()

	return m, nil
}

// startWatcher 启动文件监听
func (m *Manager) startWatcher() {
	m.watcher = NewFileWatcher(m.cfg.Agent.Dir, m.cfg.Agent.ReloadInterval, func(event FileEvent) {
		switch event.Type {
		case "create":
			klog.V(6).Infof("[Manager] Loading new agent from %s", event.Path)
			if _, err := m.loader.LoadFromPath(event.Path); err != nil {
				klog.V(6).Infof("[Manager] Failed to load agent: %v", err)
			} else {
				klog.V(6).Infof("[Manager] Successfully loaded agent from %s", event.Path)
			}

		case "modify":
			agentName := guessAgentNameFromPath(event.Path)
			klog.V(6).Infof("[Manager] Reloading agent: %s", agentName)
			if _, err := m.loader.Reload(agentName); err != nil {
				klog.V(6).Infof("[Manager] Failed to reload agent: %v", err)
			} else {
				klog.V(6).Infof("[Manager] Successfully reloaded agent: %s", agentName)
				// 清除缓存
				m.clearCache(agentName)
			}

		case "delete":
			agentName := guessAgentNameFromPath(event.Path)
			klog.V(6).Infof("[Manager] Unloading agent: %s", agentName)
			if err := m.loader.Unload(agentName); err != nil {
				klog.V(6).Infof("[Manager] Failed to unload agent: %v", err)
			} else {
				klog.V(6).Infof("[Manager] Successfully unloaded agent: %s", agentName)
				// 清除缓存
				m.clearCache(agentName)
			}
		}
	})

	if err := m.watcher.Start(); err != nil {
		klog.V(6).Infof("[Manager] Warning: failed to start file watcher: %v", err)
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
	chatModel, err := NewLLMChatModel(m.cfg)
	if err != nil {
		klog.V(6).Infof("[Manager] Failed to get model '%s', using default: %v", def.Model, err)
	}

	//将Tools进行包装为Adk可用的模式
	toolProvider := ToolProvider{BasePath: m.cfg.Data.RepoDir}
	tools := make([]tool.BaseTool, 0, len(def.Tools))
	for _, toolName := range def.Tools {
		t, tErr := toolProvider.GetTool(toolName)
		if tErr != nil {
			klog.V(6).Infof("[Manager] Warning: tool '%s' not found, skipping: %v", toolName, err)
			continue
		}
		tools = append(tools, t)
	}

	// 获取技能中间件
	skillMiddleware, err := m.GetOrCreateSkillMiddleware(m.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create skill middleware: %w", err)
	}

	// 技能由 skill middleware 自动处理，它会拦截技能调用并执行

	// 构造配置
	config := &adk.ChatModelAgentConfig{
		Name:          def.Name,
		Description:   def.Description,
		Instruction:   def.Instruction,
		Model:         chatModel,
		MaxIterations: def.MaxIterations,
		Middlewares: []adk.AgentMiddleware{
			{
				AdditionalInstruction: `
                    任务执行原则：
                    1. 如果多次尝试后仍无法完成，请总结当前进度并退出
                    2. 优先返回已完成的中间结果
                `,
			},
		},
	}

	//Skill 要单独加使用提示
	if m.skillMiddlewareHaveSkills() {
		var sn = `**重要：如何使用 Skills**
  系统提供了一个 "skill" 工具，用于调用各种专业技能。使用规则：
  - 当需要使用某个 skill 时，必须调用工具名为 "skill" 的工具
  - 参数格式：{"skill": "skill-name"}
  - 例如：要使用 repo-detection skill，调用工具名为 "skill"，参数为 {"skill": "repo-detection"}
  - 绝对不要直接调用 skill 名称（如 "repo-detection"）作为工具名
  - 可用的 skills 会在 "skill" 工具的描述中列出`
		if !strings.Contains(config.Instruction, `系统提供了一个 "skill" 工具，用于调用各种专业技能。使用规则`) {
			config.Instruction = sn + config.Instruction
		}
		config.Middlewares = append(config.Middlewares, skillMiddleware)
	}

	// 如果有工具或技能，添加 ToolsConfig
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

// guessAgentNameFromPath 从路径猜测 Agent name
// 从文件名提取（去掉扩展名）
// Agent文件的命名必须跟yaml定义中的Agent.Name字段一致
func guessAgentNameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
