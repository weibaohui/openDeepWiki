package llm

import (
	"encoding/json"
)

// Tool 定义一个可供 LLM 调用的工具
// 符合 OpenAI Function Calling 格式
type Tool struct {
	Type     string       `json:"type"` // 固定为 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 工具函数定义
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  ParameterSchema `json:"parameters"`
}

// ParameterSchema 参数 JSON Schema 定义
type ParameterSchema struct {
	Type       string              `json:"type"` // 固定为 "object"
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property 单个参数属性
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"` // 可选的枚举值
}

// ToolCall LLM 返回的工具调用请求
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // 固定为 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 格式的参数字符串
}

// ToolResult 工具执行结果
type ToolResult struct {
	Content string `json:"content"`  // 执行结果内容（文本格式）
	IsError bool   `json:"is_error"` // 是否执行出错
}

// ToolHandler 工具处理函数类型
type ToolHandler func(args json.RawMessage, basePath string) (string, error)

// ChatMessage 扩展支持 ToolCalls 和 ToolCallID
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ChatRequest 扩展支持 Tools
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"` // "none", "auto", "required", 或指定 {"type": "function", "function": {"name": "my_function"}}
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatResponse 扩展支持 ToolCalls
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role       string     `json:"role"`
			Content    string     `json:"content"`
			ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
			ToolCallID string     `json:"tool_call_id,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"` // "stop", "length", "tool_calls", etc.
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// DefaultTools 返回默认的工具集合
func DefaultTools() []Tool {
	return []Tool{
		SearchFilesTool(),
		ReadFileTool(),
		SearchTextTool(),
		ExecuteBashTool(),
		CountLinesTool(),
	}
}

// SearchFilesTool 返回 search_files 工具定义
func SearchFilesTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "search_files",
			Description: "Search for files matching a glob pattern. Use ** for recursive search (e.g., '**/*.go' for all Go files). Returns a list of file paths.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {
						Type:        "string",
						Description: "Glob pattern to match files. Supports ** for recursive matching (e.g., '**/*.go', 'src/**/*.js')",
					},
					"path": {
						Type:        "string",
						Description: "Base directory to search in (optional, defaults to current directory)",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

// ReadFileTool 返回 read_file 工具定义
func ReadFileTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "read_file",
			Description: "Read the contents of a file. Can optionally read a specific range of lines. Large files will be truncated.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the file to read (relative to base directory)",
					},
					"offset": {
						Type:        "integer",
						Description: "Line number to start reading from (1-based, optional, defaults to 1)",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of lines to read (optional, defaults to 100, max 500)",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

// SearchTextTool 返回 search_text 工具定义
func SearchTextTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "search_text",
			Description: "Search for text patterns within files. Similar to grep. Returns matching files with line numbers and content snippets.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {
						Type:        "string",
						Description: "Text pattern or regular expression to search for",
					},
					"path": {
						Type:        "string",
						Description: "Base directory to search in (optional, defaults to current directory)",
					},
					"glob": {
						Type:        "string",
						Description: "File glob pattern to filter files (e.g., '*.go', '*.js', optional)",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

// ExecuteBashTool 返回 execute_bash 工具定义
func ExecuteBashTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "execute_bash",
			Description: "Execute a bash command. Only safe commands are allowed (find, grep, wc, cat, echo, ls). Dangerous commands (rm, mv, >, |, etc.) are blocked.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"command": {
						Type:        "string",
						Description: "The bash command to execute. Only safe commands are allowed.",
					},
					"timeout": {
						Type:        "integer",
						Description: "Timeout in seconds (optional, defaults to 30, max 120)",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

// CountLinesTool 返回 count_lines 工具定义
func CountLinesTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "count_lines",
			Description: "Count lines in files or directories. Similar to wc -l. Can filter by glob pattern.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to file or directory (relative to base directory)",
					},
					"pattern": {
						Type:        "string",
						Description: "File glob pattern to filter (e.g., '*.go', optional)",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}
