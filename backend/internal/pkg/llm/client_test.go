package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)

	if client.BaseURL != "https://api.example.com" {
		t.Errorf("expected BaseURL https://api.example.com, got %s", client.BaseURL)
	}
	if client.APIKey != "test-key" {
		t.Errorf("expected APIKey test-key, got %s", client.APIKey)
	}
	if client.Model != "gpt-4" {
		t.Errorf("expected Model gpt-4, got %s", client.Model)
	}
	if client.MaxTokens != 2000 {
		t.Errorf("expected MaxTokens 2000, got %d", client.MaxTokens)
	}
	if client.Client == nil {
		t.Error("expected HTTP client to be initialized")
	}
}

func TestClientChat(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}

		// 返回模拟响应
		response := ChatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role       string     `json:"role"`
					Content    string     `json:"content"`
					ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
					ToolCallID string     `json:"tool_call_id,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					}{
						Role:    "assistant",
						Content: "This is a test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)
	messages := []ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	response, err := client.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat() unexpected error: %v", err)
	}

	if response != "This is a test response" {
		t.Errorf("expected response 'This is a test response', got %s", response)
	}
}

func TestClientChatWithTools(t *testing.T) {
	// 创建模拟服务器（返回 tool_calls）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			ID:     "test-id",
			Model:  "gpt-4",
			Object: "chat.completion",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role       string     `json:"role"`
					Content    string     `json:"content"`
					ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
					ToolCallID string     `json:"tool_call_id,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					}{
						Role:    "assistant",
						Content: "",
						ToolCalls: []ToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: FunctionCall{
									Name:      "search_files",
									Arguments: `{"pattern": "*.go"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)
	messages := []ChatMessage{
		{Role: "user", Content: "Search for Go files"},
	}
	tools := DefaultTools()

	resp, err := client.ChatWithTools(context.Background(), messages, tools)
	if err != nil {
		t.Fatalf("ChatWithTools() unexpected error: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}

	choice := resp.Choices[0]
	if len(choice.Message.ToolCalls) == 0 {
		t.Error("expected tool calls in response")
	}

	if choice.Message.ToolCalls[0].Function.Name != "search_files" {
		t.Errorf("expected tool name 'search_files', got %s", choice.Message.ToolCalls[0].Function.Name)
	}
}

func TestClientChatWithToolExecution(t *testing.T) {
	callCount := 0

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var request ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var response ChatResponse

		// 第一次调用：返回 tool_calls
		if callCount == 1 {
			response = ChatResponse{
				ID:     "test-id-1",
				Model:  "gpt-4",
				Object: "chat.completion",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role       string     `json:"role"`
							Content    string     `json:"content"`
							ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
							ToolCallID string     `json:"tool_call_id,omitempty"`
						}{
							Role:    "assistant",
							Content: "",
							ToolCalls: []ToolCall{
								{
									ID:   "call_1",
									Type: "function",
									Function: FunctionCall{
										Name:      "execute_bash",
										Arguments: `{"command": "echo hello"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			}
		} else {
			// 第二次调用：返回最终响应
			response = ChatResponse{
				ID:     "test-id-2",
				Model:  "gpt-4",
				Object: "chat.completion",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Index: 0,
						Message: struct {
							Role       string     `json:"role"`
							Content    string     `json:"content"`
							ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
							ToolCallID string     `json:"tool_call_id,omitempty"`
						}{
							Role:    "assistant",
							Content: "The command output is: hello",
						},
						FinishReason: "stop",
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)
	messages := []ChatMessage{
		{Role: "user", Content: "Run a command"},
	}
	tools := DefaultTools()

	tempDir := t.TempDir()
	response, err := client.ChatWithToolExecution(context.Background(), messages, tools, tempDir)
	if err != nil {
		t.Fatalf("ChatWithToolExecution() unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	if !strings.Contains(response, "hello") {
		t.Errorf("expected response to contain 'hello', got %s", response)
	}
}

func TestClientGenerateDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role       string     `json:"role"`
					Content    string     `json:"content"`
					ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
					ToolCallID string     `json:"tool_call_id,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					}{
						Content: "Generated document content",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)
	response, err := client.GenerateDocument(context.Background(), "You are a doc generator", "Generate docs")
	if err != nil {
		t.Fatalf("GenerateDocument() unexpected error: %v", err)
	}

	if response != "Generated document content" {
		t.Errorf("expected 'Generated document content', got %s", response)
	}
}

func TestClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			Error: &struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			}{
				Message: "Invalid API key",
				Type:    "authentication_error",
				Code:    "invalid_api_key",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)
	messages := []ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.Chat(context.Background(), messages)
	if err == nil {
		t.Error("expected error for API error response")
	}

	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected error to contain 'API error', got %v", err)
	}
}

func TestClientNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role       string     `json:"role"`
					Content    string     `json:"content"`
					ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
					ToolCallID string     `json:"tool_call_id,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)

	messages := []ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.Chat(context.Background(), messages)
	if err == nil {
		t.Error("expected error for empty choices")
	}

	if !strings.Contains(err.Error(), "no response") {
		t.Errorf("expected error to contain 'no response', got %v", err)
	}
}

func TestClientChatWithToolExecutionMaxRounds(t *testing.T) {
	// 模拟总是返回 tool_calls 的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ChatResponse{
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role       string     `json:"role"`
					Content    string     `json:"content"`
					ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
					ToolCallID string     `json:"tool_call_id,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role       string     `json:"role"`
						Content    string     `json:"content"`
						ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
						ToolCallID string     `json:"tool_call_id,omitempty"`
					}{
						Role:    "assistant",
						Content: "",
						ToolCalls: []ToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: FunctionCall{
									Name:      "execute_bash",
									Arguments: `{"command": "echo test"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			APIURL:    "https://api.example.com",
			APIKey:    "test-key",
			Model:     "gpt-4",
			MaxTokens: 2000,
		},
	}
	client := NewClient(cfg)

	messages := []ChatMessage{
		{Role: "user", Content: "Test"},
	}
	tools := DefaultTools()

	tempDir := t.TempDir()
	_, err := client.ChatWithToolExecution(context.Background(), messages, tools, tempDir)
	if err == nil {
		t.Error("expected error for exceeding max rounds")
	}

	if !strings.Contains(err.Error(), "exceeded maximum tool call rounds") {
		t.Errorf("expected error about max rounds, got %v", err)
	}
}
