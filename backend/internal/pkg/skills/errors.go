package skills

import "errors"

// 预定义错误
var (
	// ErrSkillNotFound Skill 不存在
	ErrSkillNotFound = errors.New("skill not found")

	// ErrSkillAlreadyExists Skill 已存在
	ErrSkillAlreadyExists = errors.New("skill already exists")

	// ErrInvalidMetadata 元数据无效
	ErrInvalidMetadata = errors.New("invalid skill metadata")

	// ErrInvalidName name 格式错误
	ErrInvalidName = errors.New("invalid skill name")

	// ErrInvalidDescription description 格式错误
	ErrInvalidDescription = errors.New("invalid skill description")

	// ErrSkillLoadFailed 加载失败
	ErrSkillLoadFailed = errors.New("failed to load skill")

	// ErrSkillDirNotFound Skills 目录不存在
	ErrSkillDirNotFound = errors.New("skills directory not found")

	// ErrSkillMDNotFound SKILL.md 不存在
	ErrSkillMDNotFound = errors.New("SKILL.md not found")

	// ErrInvalidFrontmatter frontmatter 格式错误
	ErrInvalidFrontmatter = errors.New("invalid YAML frontmatter")

	// ErrBodyNotFound body 内容不存在
	ErrBodyNotFound = errors.New("skill body not found")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid skill config")
)
