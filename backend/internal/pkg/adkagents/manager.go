package adkagents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
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

	docRepo repository.DocumentRepository

	// 增强的模型提供者（支持多模型和自动切换）
	enhancedModelProvider *EnhancedModelProviderImpl
}

var (
	managerInstance     *Manager
	managerInstanceOnce sync.Once
	defaultDocRepoMu    sync.RWMutex
	defaultDocRepo      repository.DocumentRepository
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

// GetOrCreateInstanceWithDocRepo 获取或创建 Manager 单例，并注入文档仓储
func GetOrCreateInstanceWithDocRepo(cfg *config.Config, docRepo repository.DocumentRepository) (*Manager, error) {
	manager, err := GetOrCreateInstance(cfg)
	if err != nil {
		return nil, err
	}
	setDocumentRepository(docRepo)
	return manager, nil
}

// setDocumentRepository 设置文档仓储，用于 read_doc 工具
func setDocumentRepository(docRepo repository.DocumentRepository) {
	defaultDocRepoMu.Lock()
	defaultDocRepo = docRepo
	defaultDocRepoMu.Unlock()

	if managerInstance != nil {
		managerInstance.docRepo = docRepo
	}

	if docRepo == nil {
		klog.V(6).Infof("[Manager] 文档仓储为空，read_doc 工具不可用")
		return
	}
	klog.V(6).Infof("[Manager] 文档仓储已注入，read_doc 工具可用")
}

// getDefaultDocRepo 获取默认文档仓储
func getDefaultDocRepo() repository.DocumentRepository {
	defaultDocRepoMu.RLock()
	defer defaultDocRepoMu.RUnlock()
	return defaultDocRepo
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
		docRepo:  getDefaultDocRepo(),
	}

	// 初始加载
	results, err := loader.LoadFromDir(cfg.Agent.Dir)
	if err != nil {
		klog.Errorf("[Manager] Failed to load agents from dir %s: %v", cfg.Agent.Dir, err)
	} else {
		klog.Infof("[Manager] Loaded %d agents from %s", len(results), cfg.Agent.Dir)
		for _, r := range results {
			if r.Error != nil {
				klog.Errorf("[Manager] Failed to load agent: %v", r.Error)
			} else if r.Agent != nil {
				klog.Infof("[Manager] Registered agent: %s", r.Agent.Name)
			}
		}
	}

	// 启动热加载
	m.startWatcher()

	return m, nil
}

// SetEnhancedModelProvider 设置增强的模型提供者
func (m *Manager) SetEnhancedModelProvider(provider *EnhancedModelProviderImpl) {
	m.enhancedModelProvider = provider
}

// GetEnhancedModelProvider 获取增强的模型提供者
func (m *Manager) GetEnhancedModelProvider() *EnhancedModelProviderImpl {
	return m.enhancedModelProvider
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
// 注意：返回的 Agent 实例可能被复用，如果需要全新实例请使用 CreateAgent
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

// CreateAgent 创建指定名称的全新 ADK Agent 实例（不使用缓存）
// 每次调用都会创建新的 Agent 实例，适用于需要独立执行场景
// 注意：创建的实例不会被缓存，需要调用者管理其生命周期
func (m *Manager) CreateAgent(name string) (adk.Agent, error) {
	// 从注册表获取定义
	def, err := m.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// 创建 ADK Agent（不使用缓存）
	agent, err := m.createADKAgent(def)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	return agent, nil
}

// createADKAgent 根据 AgentDefinition 创建 ADK Agent
func (m *Manager) createADKAgent(def *AgentDefinition) (adk.Agent, error) {
	ctx := context.Background()

	// 获取模型（支持模型池）
	var chatModel model.ToolCallingChatModel
	var err error

	if def.UseModelPool() && m.enhancedModelProvider != nil {
		// 使用模型池代理
		modelNames := def.GetModelNames()
		klog.V(6).Infof("[Manager] Using proxy model pool for agent %s: %v", def.Name, modelNames)
		chatModel = NewProxyChatModel(m.enhancedModelProvider, modelNames)
	} else if def.Model != "" && m.enhancedModelProvider != nil {
		// 使用单个模型（通过 EnhancedModelProvider）
		klog.V(6).Infof("[Manager] Using model %s for agent %s", def.Model, def.Name)
		cm, err := m.enhancedModelProvider.GetModel(def.Model)
		if err != nil {
			klog.Errorf("[Manager] Failed to get model %s: %v", def.Model, err)
			return nil, fmt.Errorf("failed to get model %s: %w", def.Model, err)
		}
		// 类型断言：将 ChatModel 转换为 ToolCallingChatModel
		var ok bool
		chatModel, ok = cm.(model.ToolCallingChatModel)
		if !ok {
			klog.Errorf("[Manager] Model %s does not implement ToolCallingChatModel", def.Model)
			return nil, fmt.Errorf("model %s does not implement ToolCallingChatModel", def.Model)
		}
	} else if m.enhancedModelProvider != nil {
		// 模型未指定，使用动态代理模型（自动从数据库选择）
		klog.V(6).Infof("[Manager] Model not specified for agent %s, using dynamic ProxyChatModel", def.Name)
		// 传入 nil 作为 modelNames，启用"自动选择所有可用模型"模式
		chatModel = NewProxyChatModel(m.enhancedModelProvider, nil)
	} else {
		// 没有增强提供者，返回错误
		return nil, fmt.Errorf("no enhanced model provider configured")
	}

	//将Tools进行包装为Adk可用的模式
	toolProvider := ToolProvider{
		BasePath: m.cfg.Data.RepoDir,
		SkillDir: m.cfg.Skill.Dir,
		DocRepo:  m.docRepo,
	}
	tools := make([]tool.BaseTool, 0, len(def.Tools))
	for _, toolName := range def.Tools {
		t, tErr := toolProvider.GetTool(toolName)
		if tErr != nil {
			klog.V(6).Infof("[Manager] Warning: tool '%s' not found, skipping: %v", toolName, err)
			continue
		}
		tools = append(tools, t)
	}

	// 构造配置
	config := &adk.ChatModelAgentConfig{
		Name:        def.Name,
		Description: def.Description,
		Instruction: def.Instruction,
		Model:       chatModel,
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 3,
			IsRetryAble: func(ctx context.Context, err error) bool {
				klog.V(6).Infof("[Manager] IsRetryAble check: %v", err)
				// 使用 RateLimiter 判断是否为 Rate Limit 错误
				rateLimiter := NewRateLimiter(m.enhancedModelProvider)
				if rateLimiter.IsRateLimitError(err) {
					klog.Warningf("[Manager] IsRetryAble: rate limit error detected, retrying...")
					return true
				}
				return false
			},
		},
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
	if def.MaxIterations > 0 {
		config.Instruction = config.Instruction + fmt.Sprintf("\n 最大交互次数 %d", def.MaxIterations)
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
