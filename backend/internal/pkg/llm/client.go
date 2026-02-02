package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"k8s.io/klog/v2"
)

// Client LLM 客户端
type Client struct {
	BaseURL   string
	APIKey    string
	Model     string
	MaxTokens int
	Client    *http.Client
}

// NewClient 创建新的 LLM 客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		BaseURL:   cfg.LLM.APIURL,
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		MaxTokens: cfg.LLM.MaxTokens,
		Client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Chat 发送对话请求（基础版本，向后兼容）
func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	klog.V(6).Infof("Chat 请求: model=%s, messages=%d", c.Model, len(messages))
	resp, err := c.sendRequest(ctx, ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		MaxTokens:   c.MaxTokens,
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatWithTools 发送带 Tools 的对话请求
func (c *Client) ChatWithTools(ctx context.Context, messages []ChatMessage, tools []Tool) (*ChatResponse, error) {
	klog.V(6).Infof("ChatWithTools 请求: model=%s, messages=%d, tools=%d", c.Model, len(messages), len(tools))
	return c.sendRequest(ctx, ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto", // 允许模型自动选择是否使用工具
		MaxTokens:   c.MaxTokens,
		Temperature: 0.7,
	})
}

// ChatWithToolExecution 发送对话请求并自动处理 Tool Calls
// 这个方法会处理多轮工具调用，直到获得最终文本响应
func (c *Client) ChatWithToolExecution(ctx context.Context, messages []ChatMessage, tools []Tool, basePath string) (string, error) {
	executor := NewSafeExecutor(&ExecutorConfig{})
	return c.ChatWithToolExecutionAndExecutor(ctx, messages, tools, executor, basePath)
}

// ChatWithToolExecutionAndExecutor 使用自定义执行器处理 Tool Calls
func (c *Client) ChatWithToolExecutionAndExecutor(ctx context.Context, messages []ChatMessage, tools []Tool, executor ToolExecutor, basePath string) (string, error) {
	klog.V(6).Infof("开始 ChatWithToolExecution: messages=%d, tools=%d", len(messages), len(tools))
	maxRounds := 100

	for round := 0; round < maxRounds; round++ {
		klog.V(6).Infof("Tool执行循环: round=%d/%d", round+1, maxRounds)
		// 发送请求到 LLM
		resp, err := c.ChatWithTools(ctx, messages, tools)
		if err != nil {
			return "", fmt.Errorf("LLM request failed: %w", err)
		}

		// 检查是否有错误
		if resp.Error != nil {
			return "", fmt.Errorf("API error: %s", resp.Error.Message)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from LLM")
		}

		choice := resp.Choices[0]
		message := choice.Message

		// 如果没有 Tool Calls，直接返回内容
		if len(message.ToolCalls) == 0 {
			klog.V(6).Infof("LLM 返回文本响应，对话结束")
			return message.Content, nil
		}

		klog.V(6).Infof("LLM 返回工具调用: count=%d", len(message.ToolCalls))
		klog.V(6).Infof("调用详情%v", message.ToolCalls)
		// 添加 assistant 的响应到消息历史
		messages = append(messages, ChatMessage{
			Role:      "assistant",
			Content:   message.Content,
			ToolCalls: message.ToolCalls,
		})

		// 执行所有 Tool Calls
		for _, toolCall := range message.ToolCalls {
			// 执行工具
			result, err := executor.Execute(ctx, toolCall, basePath)
			content := result.Content
			if err != nil {
				content = fmt.Sprintf("Error executing tool: %v", err)
				result.IsError = true
			}
			klog.V(6).Infof("工具执行结果%s ", content)

			// 添加 tool 结果到消息历史
			messages = append(messages, ChatMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    content,
			})
		}
	}

	return "", fmt.Errorf("exceeded maximum tool call rounds (%d)", maxRounds)
}

// GenerateDocument 生成文档（向后兼容）
func (c *Client) GenerateDocument(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	return c.Chat(ctx, messages)
}

// GenerateDocumentWithTools 使用工具生成文档
func (c *Client) GenerateDocumentWithTools(ctx context.Context, systemPrompt, userPrompt string, tools []Tool, basePath string) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	executor := NewSafeExecutor(&ExecutorConfig{})
	return c.ChatWithToolExecutionAndExecutor(ctx, messages, tools, executor, basePath)
}

// sendRequest 发送 HTTP 请求到 LLM API
func (c *Client) sendRequest(ctx context.Context, reqBody ChatRequest) (*ChatResponse, error) {
	url := c.BaseURL + "/chat/completions"
	klog.V(6).Infof("发送 LLM 请求: url=%s, model=%s", url, reqBody.Model)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	return &chatResp, nil
}
