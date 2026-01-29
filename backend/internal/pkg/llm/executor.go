package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/opendeepwiki/backend/internal/pkg/llm/tools"
)

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	MaxFileSize    int64         // 最大文件大小（字节）
	CommandTimeout time.Duration // 命令执行超时
	MaxResults     int           // 最大返回结果数
	MaxToolRounds  int           // 最大工具调用轮数
}

// DefaultExecutorConfig 返回默认配置
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		MaxFileSize:    1024 * 1024, // 1MB
		CommandTimeout: 30 * time.Second,
		MaxResults:     100,
		MaxToolRounds:  10,
	}
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	Execute(ctx context.Context, toolCall ToolCall) (ToolResult, error)
}

// SafeExecutor 安全的工具执行器实现
type SafeExecutor struct {
	basePath string
	config   *ExecutorConfig
	handlers map[string]ToolHandler
}

// NewSafeExecutor 创建安全的工具执行器
func NewSafeExecutor(basePath string, config *ExecutorConfig) *SafeExecutor {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	e := &SafeExecutor{
		basePath: basePath,
		config:   config,
		handlers: make(map[string]ToolHandler),
	}

	// 注册默认工具
	e.registerDefaultTools()

	return e
}

// registerDefaultTools 注册默认工具处理器
func (e *SafeExecutor) registerDefaultTools() {
	e.handlers["search_files"] = tools.SearchFiles
	e.handlers["read_file"] = tools.ReadFile
	e.handlers["search_text"] = tools.SearchText
	e.handlers["execute_bash"] = tools.ExecuteBash
	e.handlers["count_lines"] = tools.CountLines
}

// Execute 执行工具调用
func (e *SafeExecutor) Execute(ctx context.Context, toolCall ToolCall) (ToolResult, error) {
	// 验证工具调用格式
	if toolCall.Type != "function" {
		return ToolResult{
			Content: fmt.Sprintf("unsupported tool call type: %s", toolCall.Type),
			IsError: true,
		}, nil
	}

	// 获取工具处理器
	handler, ok := e.handlers[toolCall.Function.Name]
	if !ok {
		return ToolResult{
			Content: fmt.Sprintf("unknown tool: %s", toolCall.Function.Name),
			IsError: true,
		}, nil
	}

	// 执行工具
	result, err := handler(json.RawMessage(toolCall.Function.Arguments), e.basePath)
	if err != nil {
		return ToolResult{
			Content: err.Error(),
			IsError: true,
		}, nil
	}

	// 限制结果长度
	const maxResultLen = 10000
	if len(result) > maxResultLen {
		result = result[:maxResultLen] + fmt.Sprintf("\n... (%d more bytes truncated)", len(result)-maxResultLen)
	}

	return ToolResult{
		Content: result,
		IsError: false,
	}, nil
}

// ExecuteAll 执行多个工具调用
func (e *SafeExecutor) ExecuteAll(ctx context.Context, toolCalls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(toolCalls))
	for i, tc := range toolCalls {
		results[i], _ = e.Execute(ctx, tc)
	}
	return results
}

// ValidateToolCalls 验证工具调用是否安全
func (e *SafeExecutor) ValidateToolCalls(toolCalls []ToolCall) error {
	for _, tc := range toolCalls {
		if err := e.validateToolCall(tc); err != nil {
			return err
		}
	}
	return nil
}

// validateToolCall 验证单个工具调用
func (e *SafeExecutor) validateToolCall(toolCall ToolCall) error {
	if toolCall.Type != "function" {
		return fmt.Errorf("unsupported tool call type: %s", toolCall.Type)
	}

	if toolCall.Function.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	// 检查工具是否已注册
	if _, ok := e.handlers[toolCall.Function.Name]; !ok {
		return fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	// 验证参数是否为有效的 JSON
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}

	return nil
}

// GetAvailableTools 返回可用的工具列表
func (e *SafeExecutor) GetAvailableTools() []string {
	names := make([]string, 0, len(e.handlers))
	for name := range e.handlers {
		names = append(names, name)
	}
	return names
}

// ============ 安全验证辅助函数 ============

// ValidatePath 验证路径是否在基础路径内
func ValidatePath(basePath, targetPath string) error {
	// 清理路径
	cleanBase := filepath.Clean(basePath)
	cleanTarget := filepath.Clean(targetPath)

	// 确保目标路径以基础路径为前缀
	if !strings.HasPrefix(cleanTarget, cleanBase) {
		return fmt.Errorf("path escapes base directory: %s", targetPath)
	}

	return nil
}

// ValidateCommand 验证命令是否安全
func ValidateCommand(command string) error {
	// 危险命令黑名单
	dangerousPatterns := []string{
		`\brm\s+-[rf]`,
		`>\s*\/`,
		`;`,
		`\|\s*rm`,
		`\$\(`,
		"`",
		`&&\s*rm`,
	}

	for _, pattern := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, command)
		if matched {
			return fmt.Errorf("dangerous command detected")
		}
	}

	return nil
}


