package adkagents

import (
	"net/http"
)

// ClaudeHeaderTransport 伪装为 Claude CLI 的 HTTP Transport
type ClaudeHeaderTransport struct {
	rt http.RoundTripper
}

// NewClaudeHeaderTransport 创建一个伪装为 Claude CLI 的 Transport
func NewClaudeHeaderTransport() *ClaudeHeaderTransport {
	return &ClaudeHeaderTransport{
		rt: http.DefaultTransport,
	}
}

// RoundTrip 实现 http.RoundTripper 接口，注入 Claude 相关的 headers
func (t *ClaudeHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Claude CLI 特有的 headers
	headers := map[string]string{
		"User-Agent":                                "claude-cli/2.1.81 (external, sdk-cli)",
		"X-App":                                     "cli",
		"X-Stainless-Lang":                          "js",
		"X-Stainless-Os":                            "MacOS",
		"X-Stainless-Arch":                          "arm64",
		"X-Stainless-Runtime":                       "node",
		"X-Stainless-Runtime-Version":               "v24.3.0",
		"X-Stainless-Package-Version":               "0.74.0",
		"X-Stainless-Retry-Count":                   "0",
		"X-Stainless-Timeout":                       "600",
		"Anthropic-Version":                         "2023-06-01",
		"Anthropic-Dangerous-Direct-Browser-Access": "true",
		"Connection":                                "keep-alive",
		"Anthropic-Beta":                            "claude-code-20250219,interleaved-thinking-2025-05-14,prompt-caching-scope-2026-01-05,effort-2025-11-24",
	}

	for k, v := range headers {
		req.Header.Set(k, v) // 强制覆盖，确保 Claude Code 伪装生效
	}

	return t.rt.RoundTrip(req)
}

// NewClaudeHTTPClient 创建伪装为 Claude CLI 的 HTTP Client
func NewClaudeHTTPClient() *http.Client {
	return &http.Client{
		Transport: NewClaudeHeaderTransport(),
	}
}
