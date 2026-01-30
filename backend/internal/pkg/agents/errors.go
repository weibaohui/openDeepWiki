package agents

import "errors"

// 预定义错误
var (
	// ErrAgentNotFound Agent 不存在
	ErrAgentNotFound = errors.New("agent not found")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid agent config")

	// ErrInvalidName name 格式错误
	ErrInvalidName = errors.New("invalid agent name")

	// ErrAgentLoadFailed 加载失败
	ErrAgentLoadFailed = errors.New("failed to load agent")

	// ErrAgentDirNotFound Agents 目录不存在
	ErrAgentDirNotFound = errors.New("agents directory not found")

	// ErrConfigNotFound 配置文件不存在
	ErrConfigNotFound = errors.New("agent config file not found")
)
