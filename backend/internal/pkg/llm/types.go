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

// DefaultTools 返回默认的工具集合（包含所有内置工具）
func DefaultTools() []Tool {
	return []Tool{
		// 基础工具
		SearchFilesTool(),
		ReadFileTool(),
		SearchTextTool(),
		ExecuteBashTool(),
		CountLinesTool(),
		// Git 工具
		GitCloneTool(),
		GitDiffTool(),
		GitLogTool(),
		GitStatusTool(),
		GitBranchListTool(),
		// Filesystem 扩展工具
		ListDirTool(),
		FileStatTool(),
		FileExistsTool(),
		FindFilesTool(),
		// Code Analysis 工具
		ExtractFunctionsTool(),
		GetCodeSnippetTool(),
		GetFileTreeTool(),
		CalculateComplexityTool(),
		FindDefinitionsTool(),
		// Advanced Search 工具
		SemanticSearchTool(),
		SymbolSearchTool(),
		SimilarCodeTool(),
		FullTextSearchTool(),
		// Generation 工具
		GenerateMermaidTool(),
		GenerateDiagramTool(),
		SummarizeTool(),
		// Quality 工具
		CheckLinksTool(),
		CheckFormattingTool(),
		ReadabilityScoreTool(),
		SpellCheckTool(),
	}
}

// ============ 基础工具定义 ============

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
					"pattern": {Type: "string", Description: "Glob pattern to match files. Supports ** for recursive matching"},
					"path":    {Type: "string", Description: "Base directory to search in (optional)"},
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
					"path":   {Type: "string", Description: "Path to the file to read"},
					"offset": {Type: "integer", Description: "Line number to start from (1-based, optional)"},
					"limit":  {Type: "integer", Description: "Maximum lines to read (optional, max 500)"},
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
			Description: "Search for text patterns within files. Similar to grep. Returns matching files with line numbers.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {Type: "string", Description: "Text pattern or regex to search for"},
					"path":    {Type: "string", Description: "Base directory (optional)"},
					"glob":    {Type: "string", Description: "File pattern filter (optional, e.g., '*.go')"},
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
			Description: "Execute a bash command. Only safe commands allowed (find, grep, wc, cat, echo, ls).",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"command": {Type: "string", Description: "Bash command to execute"},
					"timeout": {Type: "integer", Description: "Timeout in seconds (optional, max 120)"},
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
			Description: "Count lines in files or directories. Similar to wc -l.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":    {Type: "string", Description: "Path to file or directory"},
					"pattern": {Type: "string", Description: "File pattern filter (optional)"},
				},
				Required: []string{"path"},
			},
		},
	}
}

// ============ Git 工具定义 ============

// GitCloneTool 返回 git_clone 工具定义
func GitCloneTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "git_clone",
			Description: "Clone a Git repository. Supports shallow clone with depth option.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo_url":   {Type: "string", Description: "Repository URL to clone"},
					"branch":     {Type: "string", Description: "Branch to clone (optional, defaults to main)"},
					"target_dir": {Type: "string", Description: "Target directory for the clone"},
					"depth":      {Type: "integer", Description: "Clone depth (0 = full clone)"},
				},
				Required: []string{"repo_url", "target_dir"},
			},
		},
	}
}

// GitDiffTool 返回 git_diff 工具定义
func GitDiffTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "git_diff",
			Description: "Show changes between commits or working tree.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"commit_hash": {Type: "string", Description: "Commit hash to show diff for"},
					"file_path":   {Type: "string", Description: "Specific file (optional)"},
				},
				Required: []string{"commit_hash"},
			},
		},
	}
}

// GitLogTool 返回 git_log 工具定义
func GitLogTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "git_log",
			Description: "Show commit history with hash and message.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {Type: "string", Description: "Show history for specific file (optional)"},
					"limit":     {Type: "integer", Description: "Max commits (default 10, max 50)"},
					"since":     {Type: "string", Description: "Since date (ISO 8601, optional)"},
				},
			},
		},
	}
}

// GitStatusTool 返回 git_status 工具定义
func GitStatusTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "git_status",
			Description: "Show working tree status: branch, modified and untracked files.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo_path": {Type: "string", Description: "Path to repository (optional)"},
				},
			},
		},
	}
}

// GitBranchListTool 返回 git_branch_list 工具定义
func GitBranchListTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "git_branch_list",
			Description: "List all branches in the repository.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo_path": {Type: "string", Description: "Path to repository (optional)"},
					"remote":    {Type: "boolean", Description: "List remote branches (default: false)"},
				},
			},
		},
	}
}

// ============ Filesystem 扩展工具定义 ============

// ListDirTool 返回 list_dir 工具定义
func ListDirTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "list_dir",
			Description: "List directory contents with file sizes and modification times. Supports recursive listing.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"dir":       {Type: "string", Description: "Directory path to list"},
					"recursive": {Type: "boolean", Description: "List recursively (default: false)"},
					"pattern":   {Type: "string", Description: "Glob pattern filter (optional)"},
				},
				Required: []string{"dir"},
			},
		},
	}
}

// FileStatTool 返回 file_stat 工具定义
func FileStatTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "file_stat",
			Description: "Get detailed file metadata: size, modification time, permissions.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {Type: "string", Description: "Path to file or directory"},
				},
				Required: []string{"file_path"},
			},
		},
	}
}

// FileExistsTool 返回 file_exists 工具定义
func FileExistsTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "file_exists",
			Description: "Check if a file or directory exists.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Path to check"},
				},
				Required: []string{"path"},
			},
		},
	}
}

// FindFilesTool 返回 find_files 工具定义
func FindFilesTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "find_files",
			Description: "Find files matching criteria. Supports name pattern, type filter, depth limit.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":         {Type: "string", Description: "Base directory"},
					"name_pattern": {Type: "string", Description: "Glob pattern for filename (optional)"},
					"type":         {Type: "string", Description: "Filter: 'file' or 'directory' (optional)"},
					"max_depth":    {Type: "integer", Description: "Max depth (default: 10)"},
				},
				Required: []string{"path"},
			},
		},
	}
}

// ============ Code Analysis 工具定义 ============

// ExtractFunctionsTool 返回 extract_functions 工具定义
func ExtractFunctionsTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "extract_functions",
			Description: "Extract function definitions from source code. Returns signatures, line numbers, complexity.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path":    {Type: "string", Description: "Path to source file"},
					"include_body": {Type: "boolean", Description: "Include function body (default: false)"},
				},
				Required: []string{"file_path"},
			},
		},
	}
}

// GetCodeSnippetTool 返回 get_code_snippet 工具定义
func GetCodeSnippetTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_code_snippet",
			Description: "Get a specific range of lines from a file with optional context.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path":  {Type: "string", Description: "Path to file"},
					"line_start": {Type: "integer", Description: "Starting line (1-based)"},
					"line_end":   {Type: "integer", Description: "Ending line"},
					"context":    {Type: "integer", Description: "Additional context lines (optional)"},
				},
				Required: []string{"file_path", "line_start", "line_end"},
			},
		},
	}
}

// GetFileTreeTool 返回 get_file_tree 工具定义
func GetFileTreeTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_file_tree",
			Description: "Get tree view of source files, grouped by language. Excludes vendor/node_modules.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo_path":     {Type: "string", Description: "Repository root (default: current)"},
					"include_tests": {Type: "boolean", Description: "Include test files (default: true)"},
				},
			},
		},
	}
}

// CalculateComplexityTool 返回 calculate_complexity 工具定义
func CalculateComplexityTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "calculate_complexity",
			Description: "Calculate cyclomatic complexity. Returns scores and rating (A-F).",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path":     {Type: "string", Description: "Path to source file"},
					"function_name": {Type: "string", Description: "Specific function (optional)"},
				},
				Required: []string{"file_path"},
			},
		},
	}
}

// FindDefinitionsTool 返回 find_definitions 工具定义
func FindDefinitionsTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "find_definitions",
			Description: "Find where a symbol (function, type, variable) is defined.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"symbol":    {Type: "string", Description: "Symbol name to search"},
					"repo_path": {Type: "string", Description: "Repository root (default: current)"},
					"type":      {Type: "string", Description: "Type filter: function, class, interface, variable"},
				},
				Required: []string{"symbol"},
			},
		},
	}
}

// ============ Advanced Search 工具定义 ============

// SemanticSearchTool 返回 semantic_search 工具定义
func SemanticSearchTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "semantic_search",
			Description: "Search code using semantic similarity. Returns files ranked by relevance.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":      {Type: "string", Description: "Search query"},
					"repo_path":  {Type: "string", Description: "Repository path"},
					"top_k":      {Type: "integer", Description: "Results count (default: 10, max: 20)"},
					"file_types": {Type: "array", Description: "Filter by extensions (optional)"},
				},
				Required: []string{"query"},
			},
		},
	}
}

// SymbolSearchTool 返回 symbol_search 工具定义
func SymbolSearchTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "symbol_search",
			Description: "Search for a specific symbol by exact name. Returns all occurrences.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"symbol_name": {Type: "string", Description: "Exact symbol name"},
					"repo_path":   {Type: "string", Description: "Repository path"},
					"symbol_type": {Type: "string", Description: "Type: function, class, interface, variable"},
				},
				Required: []string{"symbol_name"},
			},
		},
	}
}

// SimilarCodeTool 返回 similar_code 工具定义
func SimilarCodeTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "similar_code",
			Description: "Find code similar to a given snippet. Useful for finding duplicates.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"code_snippet": {Type: "string", Description: "Code snippet to compare"},
					"repo_path":    {Type: "string", Description: "Repository path"},
					"threshold":    {Type: "number", Description: "Similarity threshold 0-1 (default: 0.8)"},
				},
				Required: []string{"code_snippet"},
			},
		},
	}
}

// FullTextSearchTool 返回 full_text_search 工具定义
func FullTextSearchTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "full_text_search",
			Description: "Full text search with line-by-line results.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":          {Type: "string", Description: "Text to search"},
					"repo_path":      {Type: "string", Description: "Repository path"},
					"case_sensitive": {Type: "boolean", Description: "Case sensitive (default: false)"},
				},
				Required: []string{"query"},
			},
		},
	}
}

// ============ Generation 工具定义 ============

// GenerateMermaidTool 返回 generate_mermaid 工具定义
func GenerateMermaidTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "generate_mermaid",
			Description: "Generate Mermaid diagram from code. Supports flowchart, sequence, class diagrams.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"code_snippet": {Type: "string", Description: "Source code"},
					"diagram_type": {Type: "string", Description: "Type: flowchart, sequence, class (default: flowchart)"},
					"title":        {Type: "string", Description: "Diagram title (optional)"},
				},
				Required: []string{"code_snippet"},
			},
		},
	}
}

// GenerateDiagramTool 返回 generate_diagram 工具定义
func GenerateDiagramTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "generate_diagram",
			Description: "Generate architecture diagram from text description.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"description":  {Type: "string", Description: "Architecture description"},
					"diagram_type": {Type: "string", Description: "Type (default: architecture)"},
				},
				Required: []string{"description"},
			},
		},
	}
}

// SummarizeTool 返回 summarize 工具定义
func SummarizeTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "summarize",
			Description: "Summarize text. Supports concise, detailed, bullet point styles.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"text":       {Type: "string", Description: "Text to summarize"},
					"max_length": {Type: "integer", Description: "Max length (default: 200)"},
					"style":      {Type: "string", Description: "Style: concise, detailed, bullet_points"},
				},
				Required: []string{"text"},
			},
		},
	}
}

// ============ Quality 工具定义 ============

// CheckLinksTool 返回 check_links 工具定义
func CheckLinksTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "check_links",
			Description: "Check link validity in markdown. Validates internal and external links.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_content":    {Type: "string", Description: "Document content"},
					"base_path":      {Type: "string", Description: "Base path for relative links"},
					"check_external": {Type: "boolean", Description: "Check HTTP links (default: false)"},
				},
				Required: []string{"doc_content", "base_path"},
			},
		},
	}
}

// CheckFormattingTool 返回 check_formatting 工具定义
func CheckFormattingTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "check_formatting",
			Description: "Check document formatting: heading hierarchy, whitespace, code blocks.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"content": {Type: "string", Description: "Document content"},
					"format":  {Type: "string", Description: "Format: markdown (default)"},
				},
				Required: []string{"content"},
			},
		},
	}
}

// ReadabilityScoreTool 返回 readability_score 工具定义
func ReadabilityScoreTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "readability_score",
			Description: "Calculate readability: Flesch-Kincaid grade level and reading ease.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"text":   {Type: "string", Description: "Text to analyze"},
					"metric": {Type: "string", Description: "Metric: flesch_kincaid (default)"},
				},
				Required: []string{"text"},
			},
		},
	}
}

// SpellCheckTool 返回 spell_check 工具定义
func SpellCheckTool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "spell_check",
			Description: "Check spelling. Identifies common typos and repeated words.",
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]Property{
					"text":     {Type: "string", Description: "Text to check"},
					"language": {Type: "string", Description: "Language code (default: en_US)"},
				},
				Required: []string{"text"},
			},
		},
	}
}
