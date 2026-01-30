package agents

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader Agent 加载器
type Loader struct {
	parser   *Parser
	registry Registry
	mu       sync.RWMutex
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry Registry) *Loader {
	return &Loader{
		parser:   parser,
		registry: registry,
	}
}

// LoadFromDir 从目录加载所有 Agents
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error) {
	dir = filepath.Clean(dir)

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Agents directory does not exist: %s", dir)
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	results := make([]*LoadResult, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理 .yaml, .yml 和 .json 文件
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		configPath := filepath.Join(dir, entry.Name())
		result := l.loadAgent(configPath)
		results = append(results, result)
	}

	return results, nil
}

// LoadFromPath 加载单个 Agent
func (l *Loader) LoadFromPath(path string) (*Agent, error) {
	result := l.loadAgent(path)
	if result.Error != nil {
		return nil, result.Error
	}
	return result.Agent, nil
}

// loadAgent 加载 Agent（内部）
func (l *Loader) loadAgent(path string) *LoadResult {
	agent, err := l.parser.Parse(path)
	if err != nil {
		return &LoadResult{
			Error:  err,
			Action: "failed",
		}
	}

	// 检查是否已存在
	existing, _ := l.registry.Get(agent.Name)
	action := "created"
	if existing != nil {
		action = "updated"
	}

	// 注册到 Registry
	if err := l.registry.Register(agent); err != nil {
		return &LoadResult{
			Agent:  agent,
			Error:  err,
			Action: "failed",
		}
	}

	return &LoadResult{
		Agent:  agent,
		Action: action,
	}
}

// Unload 卸载 Agent
func (l *Loader) Unload(name string) error {
	return l.registry.Unregister(name)
}

// Reload 重新加载 Agent
func (l *Loader) Reload(name string) (*Agent, error) {
	agent, err := l.registry.Get(name)
	if err != nil {
		return nil, err
	}

	l.Unload(name)
	return l.LoadFromPath(agent.Path)
}
