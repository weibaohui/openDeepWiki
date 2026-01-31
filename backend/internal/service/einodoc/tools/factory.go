package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/opendeepwiki/backend/internal/pkg/llm"
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

func CreateToolsX(basePath string) []*schema.ToolInfo {
	tools := []tool.BaseTool{
		NewGitCloneTool(basePath),
		NewListDirTool(basePath),
		NewReadFileTool(basePath),
		NewSearchFilesTool(basePath),
	}

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

func CreateToolsY(basePath string) []*schema.ToolInfo {
	defaultTools := llm.DefaultTools()

	return convert(defaultTools)
}

func convert(llmTools []llm.Tool) []*schema.ToolInfo {
	toolInfos := make([]*schema.ToolInfo, 0, len(llmTools))
	for _, t := range llmTools {
		params := make(map[string]*schema.ParameterInfo)
		for name, prop := range t.Function.Parameters.Properties {
			pType := schema.String
			switch prop.Type {
			case "integer":
				pType = schema.Integer
			case "boolean":
				pType = schema.Boolean
			case "number":
				pType = schema.Number
			case "array":
				pType = schema.Array
			case "object":
				pType = schema.Object
			}

			params[name] = &schema.ParameterInfo{
				Type: pType,
				Desc: prop.Description,
			}
		}

		// 处理 Required 字段，将其追加到描述中
		for _, req := range t.Function.Parameters.Required {
			if p, ok := params[req]; ok {
				p.Desc = "(Required) " + p.Desc
			}
		}

		toolInfos = append(toolInfos, &schema.ToolInfo{
			Name:        t.Function.Name,
			Desc:        t.Function.Description,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		})
	}
	return toolInfos
}
