package llm

import (
	"encoding/json"
	"testing"
)

func TestDefaultTools(t *testing.T) {
	tools := DefaultTools()

	// DefaultTools 现在返回所有 30 个工具（包括基础、Git、Filesystem、Code Analysis、Advanced Search、Generation、Quality）
	if len(tools) != 30 {
		t.Errorf("expected 30 default tools, got %d", len(tools))
	}

	// 只检查基础工具是否存在
	expectedBasicTools := map[string]bool{
		"search_files": false,
		"read_file":    false,
		"search_text":  false,
		"execute_bash": false,
		"count_lines":  false,
	}

	for _, tool := range tools {
		if _, ok := expectedBasicTools[tool.Function.Name]; ok {
			expectedBasicTools[tool.Function.Name] = true
		}
	}

	for name, found := range expectedBasicTools {
		if !found {
			t.Errorf("expected basic tool not found: %s", name)
		}
	}
}

func TestSearchFilesTool(t *testing.T) {
	tool := SearchFilesTool()

	if tool.Type != "function" {
		t.Errorf("expected type 'function', got %s", tool.Type)
	}

	if tool.Function.Name != "search_files" {
		t.Errorf("expected name 'search_files', got %s", tool.Function.Name)
	}

	// 检查参数
	if tool.Function.Parameters.Type != "object" {
		t.Errorf("expected parameters type 'object', got %s", tool.Function.Parameters.Type)
	}

	// 检查必需参数
	hasRequiredPattern := false
	for _, req := range tool.Function.Parameters.Required {
		if req == "pattern" {
			hasRequiredPattern = true
			break
		}
	}
	if !hasRequiredPattern {
		t.Error("expected 'pattern' to be required")
	}

	// 检查属性
	if _, ok := tool.Function.Parameters.Properties["pattern"]; !ok {
		t.Error("expected 'pattern' property")
	}
	if _, ok := tool.Function.Parameters.Properties["path"]; !ok {
		t.Error("expected 'path' property")
	}
}

func TestReadFileTool(t *testing.T) {
	tool := ReadFileTool()

	if tool.Function.Name != "read_file" {
		t.Errorf("expected name 'read_file', got %s", tool.Function.Name)
	}

	// 检查必需参数
	hasRequiredPath := false
	for _, req := range tool.Function.Parameters.Required {
		if req == "path" {
			hasRequiredPath = true
			break
		}
	}
	if !hasRequiredPath {
		t.Error("expected 'path' to be required")
	}

	// 检查可选参数
	if _, ok := tool.Function.Parameters.Properties["offset"]; !ok {
		t.Error("expected 'offset' property")
	}
	if _, ok := tool.Function.Parameters.Properties["limit"]; !ok {
		t.Error("expected 'limit' property")
	}
}

func TestSearchTextTool(t *testing.T) {
	tool := SearchTextTool()

	if tool.Function.Name != "search_text" {
		t.Errorf("expected name 'search_text', got %s", tool.Function.Name)
	}

	if _, ok := tool.Function.Parameters.Properties["pattern"]; !ok {
		t.Error("expected 'pattern' property")
	}
	if _, ok := tool.Function.Parameters.Properties["glob"]; !ok {
		t.Error("expected 'glob' property")
	}
}

func TestExecuteBashTool(t *testing.T) {
	tool := ExecuteBashTool()

	if tool.Function.Name != "execute_bash" {
		t.Errorf("expected name 'execute_bash', got %s", tool.Function.Name)
	}

	// 检查必需参数
	hasRequiredCommand := false
	for _, req := range tool.Function.Parameters.Required {
		if req == "command" {
			hasRequiredCommand = true
			break
		}
	}
	if !hasRequiredCommand {
		t.Error("expected 'command' to be required")
	}

	// 检查可选参数
	if _, ok := tool.Function.Parameters.Properties["timeout"]; !ok {
		t.Error("expected 'timeout' property")
	}
}

func TestCountLinesTool(t *testing.T) {
	tool := CountLinesTool()

	if tool.Function.Name != "count_lines" {
		t.Errorf("expected name 'count_lines', got %s", tool.Function.Name)
	}

	if _, ok := tool.Function.Parameters.Properties["path"]; !ok {
		t.Error("expected 'path' property")
	}
	if _, ok := tool.Function.Parameters.Properties["pattern"]; !ok {
		t.Error("expected 'pattern' property")
	}
}

func TestToolJSONSerialization(t *testing.T) {
	tool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"param1": {
						Type:        "string",
						Description: "First parameter",
					},
				},
				Required: []string{"param1"},
			},
		},
	}

	// 序列化
	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("failed to marshal tool: %v", err)
	}

	// 反序列化
	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal tool: %v", err)
	}

	if decoded.Type != tool.Type {
		t.Errorf("expected type %s, got %s", tool.Type, decoded.Type)
	}
	if decoded.Function.Name != tool.Function.Name {
		t.Errorf("expected name %s, got %s", tool.Function.Name, decoded.Function.Name)
	}
}

func TestChatMessageJSON(t *testing.T) {
	msg := ChatMessage{
		Role:    "assistant",
		Content: "Hello",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: FunctionCall{
					Name:      "test",
					Arguments: "{}",
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	var decoded ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("expected role %s, got %s", msg.Role, decoded.Role)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(decoded.ToolCalls))
	}
}

func TestChatRequestJSON(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Tools: []Tool{
			SearchFilesTool(),
		},
		ToolChoice:  "auto",
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// 验证 JSON 包含预期的字段
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to raw map: %v", err)
	}

	if raw["model"] != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %v", raw["model"])
	}

	if _, ok := raw["tools"]; !ok {
		t.Error("expected 'tools' field in JSON")
	}

	if _, ok := raw["tool_choice"]; !ok {
		t.Error("expected 'tool_choice' field in JSON")
	}
}

func TestChatResponseJSON(t *testing.T) {
	jsonData := `{
		"id": "test-id",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello",
				"tool_calls": [{
					"id": "call_1",
					"type": "function",
					"function": {
						"name": "test",
						"arguments": "{}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		}
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != "test-id" {
		t.Errorf("expected id 'test-id', got %s", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(resp.Choices[0].Message.ToolCalls))
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected total tokens 30, got %d", resp.Usage.TotalTokens)
	}
}

func TestToolResultJSON(t *testing.T) {
	result := ToolResult{
		Content: "Success",
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if decoded.Content != result.Content {
		t.Errorf("expected content %s, got %s", result.Content, decoded.Content)
	}
	if decoded.IsError != result.IsError {
		t.Errorf("expected IsError %v, got %v", result.IsError, decoded.IsError)
	}
}
