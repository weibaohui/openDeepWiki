package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// SearchFilesTool 文件搜索工具
// 实现 Eino 的 tool.BaseTool 接口，用于搜索匹配的文件
type SearchFilesTool struct {
	basePath string // 基础路径
}

// NewSearchFilesTool 创建文件搜索工具
// basePath: 操作的基础路径
func NewSearchFilesTool(basePath string) *SearchFilesTool {
	klog.V(6).Infof("[SearchFilesTool] 创建工具实例: basePath=%s", basePath)
	return &SearchFilesTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *SearchFilesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[SearchFilesTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "search_files",
		Desc: "Search for files matching a pattern",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {
				Type: schema.String,
				Desc: "Glob pattern to match files",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 搜索匹配的文件
// 注意: 工具调用的输入输出日志由 EinoCallbacks 处理，此处仅记录业务相关日志
func (t *SearchFilesTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	var args struct {
		Pattern string `json:"pattern"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[SearchFilesTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	klog.V(6).Infof("[SearchFilesTool] 搜索文件: pattern=%s", args.Pattern)

	searchArgs, _ := json.Marshal(SearchFilesArgs{
		Pattern: args.Pattern,
	})

	result, err := SearchFiles(searchArgs, t.basePath)
	if err != nil {
		klog.Errorf("[SearchFilesTool] 搜索文件失败: %v", err)
		// 将错误信息作为字符串返回给大模型，而不是返回 error 中断节点执行
		return fmt.Sprintf("Error: %v", err), nil
	}

	return result, nil
}
