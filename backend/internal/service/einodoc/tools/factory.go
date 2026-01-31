package tools

import (
	"github.com/cloudwego/eino/components/tool"
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
