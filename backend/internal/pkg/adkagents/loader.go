package adkagents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

// Loader ADK Agent 配置加载器
type Loader struct {
	parser   *Parser
	registry *Registry
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry *Registry) *Loader {
	return &Loader{
		parser:   parser,
		registry: registry,
	}
}

// LoadFromDir 从目录加载所有 Agent 配置
// 遍历目录中的所有 .yaml 和 .yml 文件
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error) {
	results := make([]*LoadResult, 0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return results, nil // 目录不存在，返回空列表
		}
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 只处理 .yaml 和 .yml 文件
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		agent, err := l.LoadFromPath(path)

		result := &LoadResult{
			Action: "created",
		}

		if err != nil {
			result.Error = err
			result.Action = "failed"
			klog.V(6).Infof("[Loader] Failed to load agent from %s: %v", path, err)
		} else {
			result.Agent = agent
			klog.V(6).Infof("[Loader] Successfully loaded agent: %s from %s", agent.Name, path)
		}

		results = append(results, result)
	}

	return results, nil
}

// LoadFromPath 加载单个 Agent 配置
// 如果 Agent 已存在，会更新为新的配置
func (l *Loader) LoadFromPath(path string) (*AgentDefinition, error) {
	agent, err := l.parser.Parse(path)
	if err != nil {
		return nil, err
	}

	// 检查是否已存在
	exists := l.registry.Exists(agent.Name)

	// 注册到注册表
	if err := l.registry.Register(agent); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	if exists {
		klog.V(6).Infof("[Loader] Updated agent: %s", agent.Name)
	}

	return agent, nil
}

// Reload 重新加载指定 Agent
// 根据 Agent 名称从注册表中获取路径，重新加载
func (l *Loader) Reload(name string) (*AgentDefinition, error) {
	// 获取现有 Agent 的路径
	existing, err := l.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// 重新加载
	agent, err := l.parser.Parse(existing.Path)
	if err != nil {
		return nil, err
	}

	// 更新注册表
	if err := l.registry.Register(agent); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	return agent, nil
}

// Unload 卸载 Agent
func (l *Loader) Unload(name string) error {
	return l.registry.Unregister(name)
}
