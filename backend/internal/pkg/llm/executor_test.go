package llm

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNewSafeExecutor(t *testing.T) {
	basePath := "/tmp/test"
	
	// 测试默认配置
	executor := NewSafeExecutor(basePath, nil)
	if executor == nil {
		t.Fatal("NewSafeExecutor() returned nil")
	}
	if executor.basePath != basePath {
		t.Errorf("expected basePath %s, got %s", basePath, executor.basePath)
	}
	if executor.config == nil {
		t.Error("expected default config, got nil")
	}

	// 测试自定义配置
	customConfig := &ExecutorConfig{
		MaxFileSize:    2 * 1024 * 1024,
		CommandTimeout: 60 * time.Second,
		MaxResults:     200,
		MaxToolRounds:  15,
	}
	executor2 := NewSafeExecutor(basePath, customConfig)
	if executor2.config.MaxFileSize != customConfig.MaxFileSize {
		t.Errorf("expected MaxFileSize %d, got %d", customConfig.MaxFileSize, executor2.config.MaxFileSize)
	}
}

func TestSafeExecutorExecute(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewSafeExecutor(tempDir, nil)

	tests := []struct {
		name      string
		toolCall  ToolCall
		wantErr   bool
		wantCheck func(ToolResult) bool
	}{
		{
			name: "unknown tool",
			toolCall: ToolCall{
				ID:   "call_1",
				Type: "function",
				Function: FunctionCall{
					Name:      "unknown_tool",
					Arguments: "{}",
				},
			},
			wantErr: false, // 返回错误结果，不是错误
			wantCheck: func(r ToolResult) bool {
				return r.IsError && r.Content == "unknown tool: unknown_tool"
			},
		},
		{
			name: "invalid tool type",
			toolCall: ToolCall{
				ID:   "call_2",
				Type: "invalid",
				Function: FunctionCall{
					Name:      "search_files",
					Arguments: "{}",
				},
			},
			wantErr: false,
			wantCheck: func(r ToolResult) bool {
				return r.IsError
			},
		},
		{
			name: "valid search_files",
			toolCall: ToolCall{
				ID:   "call_3",
				Type: "function",
				Function: FunctionCall{
					Name:      "search_files",
					Arguments: `{"pattern": "*.go"}`,
				},
			},
			wantErr: false,
			wantCheck: func(r ToolResult) bool {
				return !r.IsError // 应该是成功的（即使没有结果）
			},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.toolCall)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantCheck != nil && !tt.wantCheck(result) {
				t.Errorf("result check failed: %+v", result)
			}
		})
	}
}

func TestSafeExecutorExecuteAll(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewSafeExecutor(tempDir, nil)

	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: FunctionCall{
				Name:      "search_files",
				Arguments: `{"pattern": "*.go"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: FunctionCall{
				Name:      "unknown_tool",
				Arguments: "{}",
			},
		},
	}

	ctx := context.Background()
	results := executor.ExecuteAll(ctx, toolCalls)

	if len(results) != len(toolCalls) {
		t.Errorf("expected %d results, got %d", len(toolCalls), len(results))
	}
}

func TestSafeExecutorValidateToolCalls(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewSafeExecutor(tempDir, nil)

	tests := []struct {
		name      string
		toolCalls []ToolCall
		wantErr   bool
	}{
		{
			name:      "empty list",
			toolCalls: []ToolCall{},
			wantErr:   false,
		},
		{
			name: "valid tool call",
			toolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "search_files",
						Arguments: `{"pattern": "*.go"}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tool type",
			toolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "invalid",
					Function: FunctionCall{
						Name:      "search_files",
						Arguments: "{}",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown tool",
			toolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "unknown",
						Arguments: "{}",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid JSON arguments",
			toolCalls: []ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "search_files",
						Arguments: "invalid json",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateToolCalls(tt.toolCalls)
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSafeExecutorGetAvailableTools(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewSafeExecutor(tempDir, nil)

	tools := executor.GetAvailableTools()
	
	expectedTools := []string{
		"search_files",
		"read_file",
		"search_text",
		"execute_bash",
		"count_lines",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %s not found in available tools", expected)
		}
	}
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()

	if config.MaxFileSize != 1024*1024 {
		t.Errorf("expected MaxFileSize %d, got %d", 1024*1024, config.MaxFileSize)
	}
	if config.CommandTimeout != 30*time.Second {
		t.Errorf("expected CommandTimeout %v, got %v", 30*time.Second, config.CommandTimeout)
	}
	if config.MaxResults != 100 {
		t.Errorf("expected MaxResults 100, got %d", config.MaxResults)
	}
	if config.MaxToolRounds != 10 {
		t.Errorf("expected MaxToolRounds 10, got %d", config.MaxToolRounds)
	}
}

func TestToolResultTruncation(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewSafeExecutor(tempDir, nil)

	// 创建一个结果会被截断的场景
	toolCall := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name: "execute_bash",
			// 生成一个大的输出
			Arguments: mustJSON(t, map[string]interface{}{
				"command": "yes | head -n 10000",
			}),
		},
	}

	ctx := context.Background()
	result, _ := executor.Execute(ctx, toolCall)

	// 结果应该被截断
	if len(result.Content) > 11000 {
		t.Errorf("result should be truncated, got length: %d", len(result.Content))
	}
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return string(b)
}
