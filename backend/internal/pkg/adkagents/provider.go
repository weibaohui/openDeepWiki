package adkagents

import (
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// SimpleModelProvider 简单的 ModelProvider 实现
type SimpleModelProvider struct {
	defaultModel model.ToolCallingChatModel
}

// NewSimpleModelProvider 创建简单的 ModelProvider
func NewSimpleModelProvider(defaultModel model.ToolCallingChatModel) *SimpleModelProvider {
	return &SimpleModelProvider{
		defaultModel: defaultModel,
	}
}

// GetModel 获取指定名称的模型
// 目前只支持默认模型，name 参数被忽略
func (p *SimpleModelProvider) GetModel(name string) (model.ToolCallingChatModel, error) {
	return p.defaultModel, nil
}

// DefaultModel 获取默认模型
func (p *SimpleModelProvider) DefaultModel() model.ToolCallingChatModel {
	return p.defaultModel
}

// SimpleToolProvider 简单的 ToolProvider 实现
type SimpleToolProvider struct {
	tools map[string]tool.BaseTool
}

// NewSimpleToolProvider 创建简单的 ToolProvider
func NewSimpleToolProvider() *SimpleToolProvider {
	return &SimpleToolProvider{
		tools: make(map[string]tool.BaseTool),
	}
}

// RegisterTool 注册工具
func (p *SimpleToolProvider) RegisterTool(name string, t tool.BaseTool) {
	p.tools[name] = t
}

// GetTool 获取指定名称的工具
func (p *SimpleToolProvider) GetTool(name string) (tool.BaseTool, error) {
	t, exists := p.tools[name]
	if exists {
		return t, nil
	}
	return nil, ErrToolNotFound
}

// ListTools 列出所有可用工具名称
func (p *SimpleToolProvider) ListTools() []string {
	names := make([]string, 0, len(p.tools))
	for name := range p.tools {
		names = append(names, name)
	}
	return names
}
