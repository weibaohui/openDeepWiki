package einodoc

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// LLMChatModel 适配现有的 llm.Client 到 Eino 的 model.ChatModel 接口
// 允许在 Eino Workflow 中使用项目已有的 LLM 客户端
type LLMChatModel struct {
	client *llm.Client // 底层 LLM 客户端
}

// NewLLMChatModel 创建 LLM ChatModel
// client: 项目已有的 llm.Client 实例
// 返回: 实现了 model.ChatModel 接口的适配器
func NewLLMChatModel(client *llm.Client) model.ChatModel {
	klog.V(6).Infof("[LLMChatModel] 创建 ChatModel 适配器")
	return &LLMChatModel{client: client}
}

// Generate 生成响应
// 实现 model.ChatModel 接口，同步生成 LLM 响应
// ctx: 上下文
// input: 消息列表
// opts: 可选参数
// 返回: 生成的消息或错误
func (m *LLMChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	klog.V(6).Infof("[LLMChatModel] Generate 开始: messageCount=%d", len(input))

	// 转换消息格式: schema.Message -> llm.ChatMessage
	messages := make([]llm.ChatMessage, len(input))
	for i, msg := range input {
		messages[i] = llm.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
		klog.V(6).Infof("[LLMChatModel]   Message[%d]: role=%s, contentLength=%d",
			i, msg.Role, len(msg.Content))
		klog.V(8).Infof("[LLMChatModel]   Message[%d]: role=%s, content=%s",
			i, msg.Role, msg.Content)
	}

	// 调用底层的 LLM 客户端
	klog.V(6).Infof("[LLMChatModel] 调用 LLM 客户端")
	response, err := m.client.Chat(ctx, messages)
	if err != nil {
		klog.Errorf("[LLMChatModel] LLM 调用失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[LLMChatModel] Generate 完成: responseLength=%d", len(response))

	// 转换回 schema.Message
	return &schema.Message{
		Role:    schema.Assistant,
		Content: response,
	}, nil
}

// Stream 流式生成
// 实现 model.ChatModel 接口，模拟流式输出
// 当前实现不支持真正的流式，将完整响应包装为单条消息的流
// ctx: 上下文
// input: 消息列表
// opts: 可选参数
// 返回: 流读取器或错误
func (m *LLMChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (
	*schema.StreamReader[*schema.Message], error) {
	klog.V(6).Infof("[LLMChatModel] Stream 开始: messageCount=%d", len(input))

	// 当前实现不支持真正的流式，使用普通生成
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		klog.Errorf("[LLMChatModel] Stream 生成失败: %v", err)
		return nil, err
	}

	// 使用 StreamReaderFromArray 创建 StreamReader
	// 将单条消息包装为流式输出
	streamReader := schema.StreamReaderFromArray([]*schema.Message{msg})
	klog.V(6).Infof("[LLMChatModel] Stream 完成: 包装为单消息流")

	return streamReader, nil
}

// BindTools 绑定工具
// 实现 model.ChatModel 接口，用于绑定可用的 Tools
// 当前实现暂不支持动态工具绑定
// tools: 工具信息列表
// 返回: 错误信息
func (m *LLMChatModel) BindTools(tools []*schema.ToolInfo) error {
	klog.V(6).Infof("[LLMChatModel] BindTools 被调用: toolCount=%d", len(tools))
	// 当前实现暂不支持动态工具绑定
	// 工具绑定可以在创建 ChatModel 时通过配置完成
	klog.V(6).Infof("[LLMChatModel] 注意: 当前实现不支持动态工具绑定")
	return nil
}

// 确保 LLMChatModel 实现了 model.ChatModel 接口
// 编译时类型检查
var _ model.ChatModel = (*LLMChatModel)(nil)
