package adkagents

import (
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/weibaohui/opendeepwiki/backend/config"
)

// AgentFactory 负责创建各种子 Agent
// 使用 adkagents.Manager 管理基础 Agent 的加载和创建
type AgentFactory struct {
	Manager  *Manager
	basePath string
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(cfg *config.Config) (*AgentFactory, error) {

	manager, err := GetOrCreateInstance(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create adkagents manager: %w", err)
	}

	return &AgentFactory{
		Manager: manager,
	}, nil
}

// GetAgent 获取指定名称的基础 Agent
// 这是获取基础 Agent 的推荐方式
func (f *AgentFactory) GetAgent(name string) (adk.Agent, error) {
	return f.Manager.GetAgent(name)
}

// Stop 停止 AgentFactory，释放资源
func (f *AgentFactory) Stop() {
	if f.Manager != nil {
		f.Manager.Stop()
	}
}
