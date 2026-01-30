package agents

import (
	"fmt"
	"sync"
)

// Registry Agent 注册中心接口
type Registry interface {
	// Register 注册 Agent
	Register(agent *Agent) error

	// Unregister 注销 Agent
	Unregister(name string) error

	// Get 获取指定名称的 Agent
	Get(name string) (*Agent, error)

	// List 列出所有已注册的 Agents
	List() []*Agent

	// Exists 检查 Agent 是否存在
	Exists(name string) bool
}

// registry Registry 的实现
type registry struct {
	mu     sync.RWMutex
	agents map[string]*Agent // name -> Agent
}

// NewRegistry 创建新的 Registry 实例
func NewRegistry() Registry {
	return &registry{
		agents: make(map[string]*Agent),
	}
}

// Register 注册 Agent
func (r *registry) Register(agent *Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	name := agent.Name
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[name] = agent
	return nil
}

// Unregister 注销 Agent
func (r *registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[name]; !exists {
		return fmt.Errorf("%w: %s", ErrAgentNotFound, name)
	}

	delete(r.agents, name)
	return nil
}

// Get 获取指定名称的 Agent
func (r *registry) Get(name string) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, name)
	}

	return agent, nil
}

// List 列出所有已注册的 Agents
func (r *registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}

	return result
}

// Exists 检查 Agent 是否存在
func (r *registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.agents[name]
	return exists
}
