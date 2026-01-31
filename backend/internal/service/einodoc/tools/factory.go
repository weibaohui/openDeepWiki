package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// CreateTools 创建工具列表
// 返回所有可用的 Eino Tools
// basePath: 工具操作的基础路径
func CreateTools(basePath string) []tool.BaseTool {
	klog.V(6).Infof("[CreateTools] 创建工具列表: basePath=%s", basePath)
	tools := []tool.BaseTool{
		NewGitCloneTool(basePath),
		NewListDirTool(basePath),
		NewReadFileTool(basePath),
		NewSearchFilesTool(basePath),
	}
	klog.V(6).Infof("[CreateTools] 工具列表创建完成: count=%d", len(tools))
	return tools
}

// CreateLLMTools 创建 LLM 工具列表
// 返回所有可用的 Eino LLM Tools
// basePath: 工具操作的基础路径
func CreateLLMTools(basePath string) []*schema.ToolInfo {
	tools := CreateTools(basePath)

	toolInfos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			klog.Errorf("[CreateToolsX] 获取工具信息失败: %v", err)
			continue
		}
		toolInfos = append(toolInfos, info)
	}
	return toolInfos
}
