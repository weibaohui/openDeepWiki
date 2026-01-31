package einodoc

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// LLMChatModel 适配现有的 llm.Client 到 Eino 的 model.ChatModel 接口
type LLMChatModel struct {
	client *llm.Client
}

// NewLLMChatModel 创建 LLM ChatModel
func NewLLMChatModel(client *llm.Client) model.ChatModel {
	return &LLMChatModel{client: client}
}

// Generate 生成响应
func (m *LLMChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 转换消息格式
	messages := make([]llm.ChatMessage, len(input))
	for i, msg := range input {
		messages[i] = llm.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 调用现有的 client
	response, err := m.client.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: response,
	}, nil
}

// Stream 流式生成（当前不支持，模拟流式返回）
func (m *LLMChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (
	*schema.StreamReader[*schema.Message], error) {
	// 当前实现不支持流式，直接返回完整响应
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// 使用 StreamReaderFromArray 创建 StreamReader
	return schema.StreamReaderFromArray([]*schema.Message{msg}), nil
}

// BindTools 绑定工具（当前不支持动态绑定）
func (m *LLMChatModel) BindTools(tools []*schema.ToolInfo) error {
	// 当前实现暂不支持动态工具绑定
	return nil
}

// 确保 LLMChatModel 实现了 model.ChatModel 接口
var _ model.ChatModel = (*LLMChatModel)(nil)
