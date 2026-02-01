package adk

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/opendeepwiki/backend/internal/service/einodoc/tools"
)

// modelProvider 实现 adkagents.ModelProvider
type modelProvider struct {
	chatModel model.ToolCallingChatModel
}

// GetModel 获取指定名称的模型，name 为空时返回默认模型
func (p *modelProvider) GetModel(name string) (model.ToolCallingChatModel, error) {
	// 目前只支持默认模型
	return p.chatModel, nil
}

// DefaultModel 获取默认模型
func (p *modelProvider) DefaultModel() model.ToolCallingChatModel {
	return p.chatModel
}

// toolProvider 实现 adkagents.ToolProvider
type toolProvider struct {
	basePath string
}

// GetTool 获取指定名称的工具
func (p *toolProvider) GetTool(name string) (tool.BaseTool, error) {
	switch name {
	case "list_dir":
		return tools.NewListDirTool(p.basePath), nil
	case "read_file":
		return tools.NewReadFileTool(p.basePath), nil
	case "search_files":
		return tools.NewSearchFilesTool(p.basePath), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ListTools 列出所有可用工具名称
func (p *toolProvider) ListTools() []string {
	return []string{"list_dir", "read_file", "search_files"}
}
