package ai

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"k8s.io/klog/v2"
)

const openAIClientName = "openai"

type OpenAIClient struct {
	nopCloser
	client      *openai.Client
	model       string
	temperature float32
	topP        float32
	tools       []openai.Tool
	maxHistory  int32
	maxTokens   int
	memory      *memoryService

	// organizationId string
}

func (c *OpenAIClient) SetTools(tools []openai.Tool) {
	c.tools = tools
}

func (c *OpenAIClient) Configure(config IAIConfig) error {
	klog.V(6).Infof("OpenAIClient Configure \n %s \n", utils.ToJSON(config))
	token := config.GetPassword()
	cfg := openai.DefaultConfig(token)
	orgId := config.GetOrganizationId()
	proxyEndpoint := config.GetProxyEndpoint()

	baseURL := config.GetBaseURL()
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	transport := &http.Transport{}
	if proxyEndpoint != "" {
		proxyUrl, err := url.Parse(proxyEndpoint)
		if err != nil {
			return err
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}

	if orgId != "" {
		cfg.OrgID = orgId
	}

	customHeaders := config.GetCustomHeaders()
	cfg.HTTPClient = &http.Client{
		Transport: &OpenAIHeaderTransport{
			Origin:  transport,
			Headers: customHeaders,
		},
	}

	client := openai.NewClientWithConfig(cfg)
	if client == nil {
		return errors.New("error creating OpenAI client")
	}
	c.client = client
	c.model = config.GetModel()
	c.temperature = config.GetTemperature()
	c.topP = config.GetTopP()
	c.maxHistory = config.GetMaxHistory()
	c.memory = NewMemoryService()
	c.maxTokens = config.GetMaxTokens()
	return nil
}

func (c *OpenAIClient) GetCompletion(ctx context.Context, contents ...any) (string, error) {
	// klog.V(6).Infof("OpenAIClient GetCompletion \n %s \n", utils.ToJSON(contents))
	c.fillChatHistory(ctx, contents)

	// Create a completion request
	resp, err := c.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:    c.model,
			Messages: c.GetHistory(ctx),
		})
	if err != nil {
		return "", err
	}
	klog.V(6).Infof("OpenAIClient GetCompletion : %s", resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content, nil
}
func (c *OpenAIClient) GetCompletionNoHistory(ctx context.Context, contents ...any) (string, error) {
	if len(contents) == 0 {
		return "", errors.New("no content provided")
	}
	// Create a completion request
	resp, err := c.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: contents[0].(string),
				},
			},
		})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
func (c *OpenAIClient) GetCompletionWithTools(ctx context.Context, contents ...any) ([]openai.ToolCall, string, error) {

	// Create a completion request
	c.fillChatHistory(ctx, contents)
	resp, err := c.client.CreateChatCompletion(ctx,
		openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    c.GetHistory(ctx),
			Temperature: c.temperature,
			TopP:        c.topP,
			Tools:       c.tools,
		})
	if err != nil {
		return nil, "", err
	}
	return resp.Choices[0].Message.ToolCalls, resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) GetStreamCompletion(ctx context.Context, contents ...any) (*openai.ChatCompletionStream, error) {
	c.fillChatHistory(ctx, contents)
	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    c.GetHistory(ctx),
		Temperature: c.temperature,
		TopP:        c.topP,
		Stream:      true,
	})
	return stream, err
}
func (c *OpenAIClient) GetStreamCompletionWithTools(ctx context.Context, contents ...any) (*openai.ChatCompletionStream, error) {
	c.fillChatHistory(ctx, contents)
	history := c.GetHistory(ctx)
	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:     c.model,
		Messages:  history,
		Tools:     c.tools,
		Stream:    true,
		MaxTokens: c.maxTokens,
	})
	klog.V(6).Infof("GetStreamCompletionWithTools 携带 history length: %d", len(history))
	klog.V(6).Infof("GetStreamCompletionWithTools c.history: %v", utils.ToJSON(history))
	return stream, err
}

func (c *OpenAIClient) GetName() string {
	return openAIClientName
}

// OpenAIHeaderTransport is an http.RoundTripper that adds the given headers to each request.
type OpenAIHeaderTransport struct {
	Origin  http.RoundTripper
	Headers []http.Header
}

// RoundTrip implements the http.RoundTripper interface.
func (t *OpenAIHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original request
	clonedReq := req.Clone(req.Context())
	for _, header := range t.Headers {
		for key, values := range header {
			// Possible values per header:  RFC 2616
			for _, value := range values {
				clonedReq.Header.Add(key, value)
			}
		}
	}

	return t.Origin.RoundTrip(clonedReq)
}

// SearchHistory 检索历史中是否包含指定内容，并返回匹配的消息内容（如有）。
func (c *OpenAIClient) SearchHistory(ctx context.Context, keyword string) (string, bool) {
	history := c.GetHistory(ctx)
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Content != "" && keyword != "" && strings.Contains(msg.Content, keyword) {
			return msg.Content, true
		}
	}
	return "", false
}
