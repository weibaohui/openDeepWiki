package einodoc

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// LLMChatModel 封装 Eino 原生的 OpenAI ChatModel
// 直接使用 cloudwego/eino-ext/components/model/openai 实现
type LLMChatModel struct {
	chatModel model.ToolCallingChatModel // 底层 OpenAI ChatModel 实例
}

// NewLLMChatModel 创建 LLM ChatModel
// apiKey: OpenAI API Key
// baseURL: API 基础 URL (可选，为空时使用默认 OpenAI URL)
// modelName: 模型名称 (如 "gpt-4o", "gpt-3.5-turbo" 等)
// maxTokens: 最大生成 token 数
// 返回: 实现了 model.ToolCallingChatModel 接口的实例
func NewLLMChatModel(apiKey, baseURL, modelName string, maxTokens int) (*LLMChatModel, error) {
	klog.V(6).Infof("[LLMChatModel] 创建 OpenAI ChatModel: model=%s, baseURL=%s", modelName, baseURL)

	config := &openai.ChatModelConfig{
		APIKey: apiKey,
		Model:  modelName,
	}

	if baseURL != "" {
		config.BaseURL = baseURL
	}

	if maxTokens > 0 {
		config.MaxTokens = &maxTokens
	}

	chatModel, err := openai.NewChatModel(context.Background(), config)
	if err != nil {
		klog.Errorf("[LLMChatModel] 创建 ChatModel 失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[LLMChatModel] ChatModel 创建成功")
	cm := &LLMChatModel{chatModel: chatModel}
	if err != nil {
		klog.Errorf("[LLMChatModel] 设置工具失败: %v", err)
		return nil, err
	}
	return cm, nil
}

// Generate 生成响应
// 实现 model.ChatModel 接口，同步生成 LLM 响应
// ctx: 上下文
// input: 消息列表
// opts: 可选参数
// 返回: 生成的消息或错误
func (m *LLMChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	klog.V(6).Infof("[LLMChatModel] Generate 开始: messageCount=%d", len(input))

	for i, msg := range input {
		klog.V(6).Infof("[LLMChatModel]   Message[%d]: role=%s, contentLength=%d", i, msg.Role, len(msg.Content))
		klog.V(8).Infof("[LLMChatModel]   Message[%d]: content=%s", i, msg.Content)
	}

	resp, err := m.chatModel.Generate(ctx, input, opts...)
	if err != nil {
		klog.Errorf("[LLMChatModel] Generate 失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[LLMChatModel] Generate 完成: responseLength=%d", len(resp.Content))
	return resp, nil
}

// Stream 流式生成
// 实现 model.ChatModel 接口
// ctx: 上下文
// input: 消息列表
// opts: 可选参数
// 返回: 流读取器或错误
func (m *LLMChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (
	*schema.StreamReader[*schema.Message], error) {
	klog.V(6).Infof("[LLMChatModel] Stream 开始: messageCount=%d", len(input))

	streamReader, err := m.chatModel.Stream(ctx, input, opts...)
	if err != nil {
		klog.Errorf("[LLMChatModel] Stream 失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[LLMChatModel] Stream 完成")
	return streamReader, nil
}

// WithTools 设置工具并返回新的 ChatModel 实例
// 实现 model.ToolCallingChatModel 接口
// tools: 要使用的工具列表
func (m *LLMChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	klog.V(6).Infof("[LLMChatModel] WithTools 被调用: toolCount=%d", len(tools))
	return m.chatModel.WithTools(tools)
}
