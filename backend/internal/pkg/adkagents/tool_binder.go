package adkagents

import (
	"sync"

	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// ToolBinder 工具绑定器
type ToolBinder struct {
	tools []*schema.ToolInfo
	mu    sync.RWMutex
}

// NewToolBinder 创建工具绑定器
func NewToolBinder() *ToolBinder {
	return &ToolBinder{
		tools: make([]*schema.ToolInfo, 0),
	}
}

// BindTools 设置要绑定的工具列表
func (b *ToolBinder) BindTools(tools []*schema.ToolInfo) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tools = tools
	return nil
}

// BindToModel 将工具绑定到指定模型
func (b *ToolBinder) BindToModel(model interface{}) error {
	b.mu.RLock()
	tools := b.tools
	b.mu.RUnlock()

	if len(tools) == 0 {
		return nil
	}

	// 使用类型断言检查模型是否支持 BindTools
	type ToolBindable interface {
		BindTools(tools []*schema.ToolInfo) error
	}

	if binder, ok := model.(ToolBindable); ok {
		if err := binder.BindTools(tools); err != nil {
			klog.Warningf("ToolBinder: failed to bind tools to model: %v", err)
		}
	} else {
		klog.V(6).Infof("ToolBinder: model does not support BindTools")
	}

	return nil
}

// GetTools 获取当前工具列表
func (b *ToolBinder) GetTools() []*schema.ToolInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.tools
}
