package skills

import "errors"

// 预定义错误
var (
	// ErrSkillNotFound Skill 不存在
	ErrSkillNotFound = errors.New("skill not found")

	// ErrSkillDisabled Skill 已禁用
	ErrSkillDisabled = errors.New("skill is disabled")

	// ErrSkillAlreadyExists Skill 已存在
	ErrSkillAlreadyExists = errors.New("skill already exists")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid skill config")

	// ErrProviderNotFound Provider 不存在
	ErrProviderNotFound = errors.New("provider not found")

	// ErrInvalidProviderType 无效的 Provider 类型
	ErrInvalidProviderType = errors.New("invalid provider type")

	// ErrExecutionFailed 执行失败
	ErrExecutionFailed = errors.New("skill execution failed")

	// ErrTimeout 执行超时
	ErrTimeout = errors.New("skill execution timeout")
)
