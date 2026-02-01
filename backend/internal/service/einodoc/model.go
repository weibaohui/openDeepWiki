package einodoc

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/opendeepwiki/backend/config"
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
func NewLLMChatModel(cfg *config.Config) (*openai.ChatModel, error) {
	config := &openai.ChatModelConfig{
		BaseURL:   cfg.LLM.APIURL,
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		MaxTokens: &cfg.LLM.MaxTokens,
	}

	chatModel, err := openai.NewChatModel(context.Background(), config)
	if err != nil {
		klog.Errorf("[LLMChatModel] 创建 ChatModel 失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[LLMChatModel] ChatModel 创建成功")

	return chatModel, nil
}
