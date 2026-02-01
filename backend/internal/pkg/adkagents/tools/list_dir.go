package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/pkg/llm/tools"
)

// ListDirTool 目录列表工具
// 实现 Eino 的 tool.BaseTool 接口，用于列出目录内容
type ListDirTool struct {
	basePath string // 基础路径
}

// NewListDirTool 创建目录列表工具
// basePath: 操作的基础路径
func NewListDirTool(basePath string) *ListDirTool {
	klog.V(6).Infof("[ListDirTool] 创建工具实例: basePath=%s", basePath)
	return &ListDirTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *ListDirTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[ListDirTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "list_dir",
		Desc: "List directory contents with file information",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"dir": {
				Type: schema.String,
				Desc: "Directory path to list",
			},
			"recursive": {
				Type: schema.Boolean,
				Desc: "List recursively",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 列出指定目录的内容
// 注意: 工具调用的输入输出日志由 EinoCallbacks 处理，此处仅记录业务相关日志
func (t *ListDirTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	var args struct {
		Dir       string `json:"dir"`
		Recursive bool   `json:"recursive"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[ListDirTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	klog.V(6).Infof("[ListDirTool] 列出目录: dir=%s, recursive=%v", args.Dir, args.Recursive)

	listArgs, _ := json.Marshal(tools.ListDirArgs{
		Dir:       args.Dir,
		Recursive: args.Recursive,
	})

	result, err := tools.ListDir(listArgs, t.basePath)
	if err != nil {
		klog.Errorf("[ListDirTool] 列出目录失败: %v", err)
		// 将错误信息作为字符串返回给大模型，而不是返回 error 中断节点执行
		return fmt.Sprintf("Error: %v", err), nil
	}

	return result, nil
}
