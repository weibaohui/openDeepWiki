package adkagents

import (
	"fmt"
	"sync"
)

// Registry Agent 定义注册表
type Registry struct {
	agents map[string]*AgentDefinition
	mutex  sync.RWMutex
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*AgentDefinition),
	}
}

// Register 注册 Agent 定义
// 如果 Agent 已存在，返回 ErrAgentAlreadyExists
func (r *Registry) Register(def *AgentDefinition) error {
	if def == nil {
		return fmt.Errorf("%w: agent definition is nil", ErrInvalidConfig)
	}

	if def.Name == "" {
		return fmt.Errorf("%w: agent name is required", ErrInvalidName)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.agents[def.Name] = def
	return nil
}

// Unregister 注销 Agent
// 如果 Agent 不存在，返回 ErrAgentNotFound
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.agents[name]; !exists {
		return fmt.Errorf("%w: %s", ErrAgentNotFound, name)
	}

	delete(r.agents, name)
	return nil
}

// Get 获取 Agent 定义
// 如果 Agent 不存在，返回 ErrAgentNotFound
func (r *Registry) Get(name string) (*AgentDefinition, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	agent, exists := r.agents[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, name)
	}

	return agent, nil
}

// List 列出所有 Agent 定义
// 返回按名称排序的 Agent 定义列表
func (r *Registry) List() []*AgentDefinition {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*AgentDefinition, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}

	return result
}

// Exists 检查 Agent 是否存在
func (r *Registry) Exists(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.agents[name]
	return exists
}

// Count 返回注册的 Agent 数量
func (r *Registry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.agents)
}

// Clear 清空所有 Agent 定义
func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.agents = make(map[string]*AgentDefinition)
}
