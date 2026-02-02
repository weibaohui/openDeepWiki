package adkagents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

// NewLLMChatModel 创建 LLM ChatModel
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
