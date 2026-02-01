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

// ReadFileTool 文件读取工具
// 实现 Eino 的 tool.BaseTool 接口，用于读取文件内容
type ReadFileTool struct {
	basePath string // 基础路径
}

// NewReadFileTool 创建文件读取工具
// basePath: 操作的基础路径
func NewReadFileTool(basePath string) *ReadFileTool {
	klog.V(6).Infof("[ReadFileTool] 创建工具实例: basePath=%s", basePath)
	return &ReadFileTool{basePath: basePath}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[ReadFileTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "Read content of a file",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type: schema.String,
				Desc: "File path to read",
			},
			"limit": {
				Type: schema.Integer,
				Desc: "Maximum lines to read (optional, default 100)",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 读取指定文件的内容
// 注意: 工具调用的输入输出日志由 EinoCallbacks 处理，此处仅记录业务相关日志
func (t *ReadFileTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	var args struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[ReadFileTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit == 0 {
		args.Limit = 100 // 默认读取 100 行
	}

	klog.V(6).Infof("[ReadFileTool] 读取文件: path=%s, limit=%d", args.Path, args.Limit)

	readArgs, _ := json.Marshal(tools.ReadFileArgs{
		Path:  args.Path,
		Limit: args.Limit,
	})

	result, err := tools.ReadFile(readArgs, t.basePath)
	if err != nil {
		klog.Errorf("[ReadFileTool] 读取文件失败: %v", err)
		return "", err
	}

	return result, nil
}
