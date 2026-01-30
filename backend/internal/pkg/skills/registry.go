package skills

import (
	"fmt"
	"sync"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// Registry Skill 注册中心接口
type Registry interface {
	// Register 注册 Skill
	// 如果同名 Skill 已存在，返回 ErrSkillAlreadyExists
	Register(skill Skill) error

	// Unregister 注销 Skill
	// 如果不存在，返回 ErrSkillNotFound
	Unregister(name string) error

	// Enable 启用 Skill
	// 只有 enabled 的 Skill 才会被暴露给 LLM
	Enable(name string) error

	// Disable 禁用 Skill
	// 禁用后 Skill 保留在 Registry 中，但不可被 LLM 调用
	Disable(name string) error

	// IsEnabled 检查 Skill 是否已启用
	IsEnabled(name string) bool

	// Get 获取指定名称的 Skill
	Get(name string) (Skill, error)

	// List 列出所有已注册的 Skills
	List() []Skill

	// ListEnabled 列出所有已启用的 Skills
	ListEnabled() []Skill

	// ToTools 将所有 enabled Skills 转换为 LLM Tools
	ToTools() []llm.Tool
}

// registry Registry 的实现
type registry struct {
	mu      sync.RWMutex
	skills  map[string]Skill // name -> Skill
	enabled map[string]bool  // name -> enabled
}

// NewRegistry 创建新的 Registry 实例
func NewRegistry() Registry {
	return &registry{
		skills:  make(map[string]Skill),
		enabled: make(map[string]bool),
	}
}

// Register 注册 Skill
func (r *registry) Register(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	name := skill.Name()
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("%w: skill %q already registered", ErrSkillAlreadyExists, name)
	}

	r.skills[name] = skill
	r.enabled[name] = true // 默认启用

	return nil
}

// Unregister 注销 Skill
func (r *registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}

	delete(r.skills, name)
	delete(r.enabled, name)

	return nil
}

// Enable 启用 Skill
func (r *registry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}

	r.enabled[name] = true
	return nil
}

// Disable 禁用 Skill
func (r *registry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}

	r.enabled[name] = false
	return nil
}

// IsEnabled 检查 Skill 是否已启用
func (r *registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.enabled[name]
}

// Get 获取指定名称的 Skill
func (r *registry) Get(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}

	return skill, nil
}

// List 列出所有已注册的 Skills
func (r *registry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		result = append(result, skill)
	}

	return result
}

// ListEnabled 列出所有已启用的 Skills
func (r *registry) ListEnabled() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Skill, 0)
	for name, skill := range r.skills {
		if r.enabled[name] {
			result = append(result, skill)
		}
	}

	return result
}

// ToTools 将所有 enabled Skills 转换为 LLM Tools
func (r *registry) ToTools() []llm.Tool {
	enabled := r.ListEnabled()
	tools := make([]llm.Tool, 0, len(enabled))

	for _, skill := range enabled {
		tool := llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        skill.Name(),
				Description: skill.Description(),
				Parameters:  skill.Parameters(),
			},
		}
		tools = append(tools, tool)
	}

	return tools
}
