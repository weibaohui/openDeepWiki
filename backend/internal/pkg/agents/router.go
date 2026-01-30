package agents

import (
	"fmt"
	"sync"
)

// Router Agent 路由器接口
type Router interface {
	// Route 根据上下文选择 Agent
	Route(ctx RouterContext) (*Agent, error)

	// SetDefault 设置默认 Agent
	SetDefault(agentName string) error

	// RegisterRoute 注册路由规则
	RegisterRoute(entryPoint string, agentName string)

	// GetRoute 获取路由规则
	GetRoute(entryPoint string) (string, bool)

	// GetDefault 获取默认 Agent
	GetDefault() string
}

// router Router 的实现
type router struct {
	registry     Registry
	defaultAgent string
	routes       map[string]string // entryPoint -> agentName
	mu           sync.RWMutex
}

// NewRouter 创建新的 Router 实例
func NewRouter(registry Registry) Router {
	return &router{
		registry: registry,
		routes:   make(map[string]string),
	}
}

// Route 根据上下文选择 Agent
func (r *router) Route(ctx RouterContext) (*Agent, error) {
	// 1. 优先级最高：显式指定 Agent name
	if ctx.AgentName != "" {
		agent, err := r.registry.Get(ctx.AgentName)
		if err != nil {
			return nil, fmt.Errorf("explicitly specified agent not found: %w", err)
		}
		return agent, nil
	}

	// 2. 根据 EntryPoint 路由
	if ctx.EntryPoint != "" {
		r.mu.RLock()
		agentName, exists := r.routes[ctx.EntryPoint]
		r.mu.RUnlock()

		if exists {
			agent, err := r.registry.Get(agentName)
			if err != nil {
				return nil, fmt.Errorf("route found but agent not found: %w", err)
			}
			return agent, nil
		}
	}

	// 3. 使用默认 Agent
	r.mu.RLock()
	defaultAgent := r.defaultAgent
	r.mu.RUnlock()

	if defaultAgent != "" {
		agent, err := r.registry.Get(defaultAgent)
		if err != nil {
			return nil, fmt.Errorf("default agent not found: %w", err)
		}
		return agent, nil
	}

	// 4. 没有任何可用 Agent
	return nil, fmt.Errorf("%w: no matching agent found", ErrAgentNotFound)
}

// SetDefault 设置默认 Agent
func (r *router) SetDefault(agentName string) error {
	if !r.registry.Exists(agentName) {
		return fmt.Errorf("%w: %s", ErrAgentNotFound, agentName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultAgent = agentName
	return nil
}

// RegisterRoute 注册路由规则
func (r *router) RegisterRoute(entryPoint string, agentName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes[entryPoint] = agentName
}

// GetRoute 获取路由规则
func (r *router) GetRoute(entryPoint string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agentName, exists := r.routes[entryPoint]
	return agentName, exists
}

// GetDefault 获取默认 Agent
func (r *router) GetDefault() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultAgent
}
