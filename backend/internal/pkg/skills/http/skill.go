package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
	"github.com/opendeepwiki/backend/internal/pkg/skills"
)

// HTTPSkill HTTP Skill 实现
type HTTPSkill struct {
	config  skills.SkillConfig
	client  *Client
	timeout time.Duration
}

// NewHTTPSkill 创建 HTTP Skill
func NewHTTPSkill(config skills.SkillConfig, client *Client) *HTTPSkill {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	return &HTTPSkill{
		config:  config,
		client:  client,
		timeout: time.Duration(timeout) * time.Second,
	}
}

// Name 返回名称
func (s *HTTPSkill) Name() string {
	return s.config.Name
}

// Description 返回描述
func (s *HTTPSkill) Description() string {
	return s.config.Description
}

// Parameters 返回参数定义
func (s *HTTPSkill) Parameters() llm.ParameterSchema {
	return s.config.Parameters
}

// Execute 执行 HTTP 调用
func (s *HTTPSkill) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.Endpoint, bytes.NewReader(args))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 Headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range s.config.Headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析响应
	var result interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			// 如果不是 JSON，返回原始字符串
			return string(body), nil
		}
	}

	return result, nil
}

// ProviderType 返回 Provider 类型
func (s *HTTPSkill) ProviderType() string {
	return "http"
}
