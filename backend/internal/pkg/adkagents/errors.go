// Package adkagents 提供 ADK Agent 的 YAML 配置化管理
package adkagents

import "errors"

// 错误定义
var (
	// ErrAgentNotFound Agent 不存在
	ErrAgentNotFound = errors.New("agent not found")

	// ErrInvalidConfig 配置文件无效
	ErrInvalidConfig = errors.New("invalid agent config")

	// ErrInvalidName Agent 名称格式无效
	ErrInvalidName = errors.New("invalid agent name")

	// ErrToolNotFound 工具不存在
	ErrToolNotFound = errors.New("tool not found")

	// ErrModelNotFound 模型不存在
	ErrModelNotFound = errors.New("model not found")

	// ErrAgentDirNotFound Agent 目录不存在
	ErrAgentDirNotFound = errors.New("agents directory not found")

	// ErrAgentAlreadyExists Agent 已存在
	ErrAgentAlreadyExists = errors.New("agent already exists")

	// ErrConfigNotFound 配置文件不存在
	ErrConfigNotFound = errors.New("config file not found")
)
