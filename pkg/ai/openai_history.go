package ai

import (
	"context"
	"fmt"
	"strings"

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

	key := c.CacheKey(ctx)
	c.memory.SetRepoHistory(key, history)

}

func (c *OpenAIClient) SaveAIHistory(ctx context.Context, contents string) {
	if contents == "" {
		return
	}
	key := c.CacheKey(ctx)
	c.memory.AppendRepoHistory(key, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: contents,
	})
}

// CacheKey 缓存 key 生成规则：repoName-userName
func (c *OpenAIClient) CacheKey(ctx context.Context) string {
	repoNameVal := ""
	if repoName, ok := ctx.Value(constants.RepoName).(string); ok {
		repoNameVal = repoName
	}
	userNameVal := ""
	if userName, ok := ctx.Value(constants.JwtUserName).(string); ok {
		userNameVal = userName
	}
	return fmt.Sprintf("%s-%s", repoNameVal, userNameVal)
}

func (c *OpenAIClient) GetHistory(ctx context.Context) []openai.ChatCompletionMessage {
	key := c.CacheKey(ctx)
	return c.memory.GetRepoHistory(key)
}

func (c *OpenAIClient) ClearHistory(ctx context.Context) error {
	key := c.CacheKey(ctx)
	c.memory.ClearRepoHistory(key)
	return nil
}

// SummarizeHistory 对历史记录进行归纳总结，排除系统提示词，通过 AI 归纳每条历史，提取关键信息，减少 token 占用。
func (c *OpenAIClient) SummarizeHistory(ctx context.Context) error {
	history := c.GetHistory(ctx)
	if len(history) == 0 {
		return nil
	}
	var summarized []openai.ChatCompletionMessage
	for _, msg := range history {
		if msg.Role == openai.ChatMessageRoleSystem {
			summarized = append(summarized, msg)
			continue
		}
		if msg.Content == "" {
			continue
		}
		// 调用 AI 进行归纳总结
		result, err := c.summarizeMessageWithAI(ctx, msg.Content)
		if err != nil {
			klog.Warningf("SummarizeHistory AI error: %v", err)
			// 失败则保留原内容
			summarized = append(summarized, msg)
			continue
		}
		summarized = append(summarized, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: result,
		})
	}
	key := c.CacheKey(ctx)
	c.memory.SetRepoHistory(key, summarized)
	return nil
}

// CheckAndSummarizeHistory 检查历史条数或内容长度，超过阈值则自动归纳总结。
func (c *OpenAIClient) CheckAndSummarizeHistory(ctx context.Context, maxCount int, maxTotalLen int) error {
	// TODO 轮数、token数做成配置项
	history := c.GetHistory(ctx)
	count := 0
	totalLen := 0
	for _, msg := range history {
		if msg.Role == openai.ChatMessageRoleSystem {
			continue
		}
		count++
		totalLen += len(msg.Content)
	}
	if (maxCount > 0 && count > maxCount) || (maxTotalLen > 0 && totalLen > maxTotalLen) {
		return c.SummarizeHistory(ctx)
	}
	return nil
}

// summarizeMessageWithAI 使用 AI 对单条消息内容进行归纳总结。
func (c *OpenAIClient) summarizeMessageWithAI(ctx context.Context, content string) (string, error) {
	if content == "" {
		return content, nil
	}
	if strings.Contains(content, "<已归纳>") {
		return content, nil
	}
	klog.V(2).Infof("Summarizing message with AI: %v", content)
	// 这里假设有一个 SummarizePrompt 作为归纳指令
	prompt := "请对以下内容进行归纳总结，提取有用的关键信息，避免冗余。 ：" + content
	resp, err := c.GetCompletionNoHistory(ctx, prompt)
	if err != nil {
		return content, err
	}
	resp = "<已归纳> " + resp
	klog.V(2).Infof("Summarized message: %v", resp)
	return resp, nil
}
