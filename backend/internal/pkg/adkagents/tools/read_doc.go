package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// ReadDocTool 文档读取工具
// 实现 Eino 的 tool.BaseTool 接口，用于按文档ID读取全文
type ReadDocTool struct {
	docRepo repository.DocumentRepository
}

// NewReadDocTool 创建文档读取工具
// docRepo: 文档仓储
func NewReadDocTool(docRepo repository.DocumentRepository) *ReadDocTool {
	klog.V(6).Infof("[ReadDocTool] 创建工具实例")
	return &ReadDocTool{docRepo: docRepo}
}

// Info 返回工具信息
// 实现 tool.BaseTool 接口
func (t *ReadDocTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	klog.V(6).Infof("[ReadDocTool] 获取工具信息")
	return &schema.ToolInfo{
		Name: "read_doc",
		Desc: "Read full document content by document ID",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"doc_id": {
				Type: schema.Integer,
				Desc: "Document ID to read",
			},
		}),
	}, nil
}

// InvokableRun 执行工具调用
// 读取指定文档ID的全文内容
func (t *ReadDocTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	if t.docRepo == nil {
		klog.Errorf("[ReadDocTool] 文档仓储未初始化")
		return "Error: 文档仓储未初始化", nil
	}

	var args struct {
		DocID uint `json:"doc_id"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		klog.Errorf("[ReadDocTool] 参数解析失败: %v", err)
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.DocID == 0 {
		klog.Errorf("[ReadDocTool] 参数校验失败: doc_id 不能为空")
		return "Error: doc_id 不能为空", nil
	}

	klog.V(6).Infof("[ReadDocTool] 读取文档: doc_id=%d", args.DocID)

	doc, err := t.docRepo.Get(args.DocID)
	if err != nil {
		klog.Errorf("[ReadDocTool] 读取文档失败: doc_id=%d, error=%v", args.DocID, err)
		return fmt.Sprintf("Error: %v", err), nil
	}

	return doc.Content, nil
}
