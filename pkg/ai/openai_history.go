package ai

import (
	"context"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/sashabaranov/go-openai"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"k8s.io/klog/v2"
)

func (c *OpenAIClient) fillChatHistory(ctx context.Context, contents ...any) {

	history := c.GetHistory(ctx)
	for _, content := range contents {
		switch item := content.(type) {
		case string:
			klog.V(2).Infof("Adding user message to history: %v", item)
			history = append(history, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: item,
			})
		case models.MCPToolCallResult:
			klog.V(2).Infof("Adding user message to history: %v", item)
			history = append(history, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: utils.ToJSON(item),
			})
		case []string:
			klog.V(2).Infof("Adding string array to history: %v", item)
			for _, m := range item {
				history = append(history, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: m,
				})
			}
		case []models.MCPToolCallResult:
			klog.V(2).Infof("Adding MCPToolCallResult array to history: %v", item)
			for _, m := range item {
				history = append(history, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: utils.ToJSON(m),
				})
			}
		case []interface{}:
			for _, m := range item {
				history = append(history, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: utils.ToJSON(m),
				})
			}
		default:
			klog.Warningf("Unhandled content type in Send: %T", item)
		}
	}

	// 保留最后 maxHistory 条（含系统提示）
	if c.maxHistory > 0 && int32(len(history)) > c.maxHistory {
		keep := history[len(history)-int(c.maxHistory):]
		history = keep
	}
	systemPrompt := ctx.Value(constants.SystemPrompt).(string)
	if systemPrompt != "" {
		system := slice.Filter(history, func(index int, item openai.ChatCompletionMessage) bool {
			return item.Role == openai.ChatMessageRoleSystem
		})

		if len(system) == 0 {
			// 创建系统消息
			sysMsg := openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			}
			// 将系统消息插入到历史记录最前面
			history = append([]openai.ChatCompletionMessage{sysMsg}, history...)
		}
	}

	repoName := ctx.Value(constants.RepoName).(string)
	c.memory.SetRepoHistory(repoName, history)

}

func (c *OpenAIClient) SaveAIHistory(ctx context.Context, contents string) {
	val := ctx.Value(constants.RepoName)
	if repoName, ok := val.(string); ok {
		c.memory.AppendRepoHistory(repoName, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: contents,
		})
	} else {
		klog.Warningf("SaveAIHistory content but repo not found: %s", contents)
	}
}

func (c *OpenAIClient) GetHistory(ctx context.Context) []openai.ChatCompletionMessage {
	val := ctx.Value(constants.RepoName)
	if repoName, ok := val.(string); ok {
		return c.memory.GetRepoHistory(repoName)
	}
	return make([]openai.ChatCompletionMessage, 0)
}

func (c *OpenAIClient) ClearHistory(ctx context.Context) error {
	val := ctx.Value(constants.RepoName)
	if repoName, ok := val.(string); ok {
		c.memory.ClearRepoHistory(repoName)
	}
	return nil
}
