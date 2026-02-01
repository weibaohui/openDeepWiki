package adkagents

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser ADK Agent 配置解析器
type Parser struct {
	maxNameLen        int
	maxDescriptionLen int
	maxInstructionLen int
}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{
		maxNameLen:        64,
		maxDescriptionLen: 1024,
		maxInstructionLen: 100 * 1024, // 100KB
	}
}

// Parse 解析 Agent 配置文件
func (p *Parser) Parse(configPath string) (*AgentDefinition, error) {
	configPath = filepath.Clean(configPath)

	// 读取文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configPath)
		}
		return nil, fmt.Errorf("failed to read agent config: %w", err)
	}

	// 解析 YAML
	agent := &AgentDefinition{}
	if err := yaml.Unmarshal(content, agent); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// 设置路径信息
	agent.Path = configPath
	agent.LoadedAt = Now()

	// 校验
	if err := p.Validate(agent); err != nil {
		return nil, err
	}

	return agent, nil
}

// Validate 校验 Agent 配置
func (p *Parser) Validate(agent *AgentDefinition) error {
	// 校验 name
	if agent.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidName)
	}
	if len(agent.Name) > p.maxNameLen {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidName, p.maxNameLen)
	}
	if !isValidAgentName(agent.Name) {
		return fmt.Errorf("%w: name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with hyphen", ErrInvalidName)
	}

	// 校验 description
	if agent.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidConfig)
	}
	if len(agent.Description) > p.maxDescriptionLen {
		return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidConfig, p.maxDescriptionLen)
	}

	// 校验 instruction
	if agent.Instruction == "" {
		return fmt.Errorf("%w: instruction is required", ErrInvalidConfig)
	}
	if len(agent.Instruction) > p.maxInstructionLen {
		return fmt.Errorf("%w: instruction exceeds %d characters", ErrInvalidConfig, p.maxInstructionLen)
	}

	// 校验 maxIterations
	if agent.MaxIterations <= 0 {
		return fmt.Errorf("%w: maxIterations must be positive", ErrInvalidConfig)
	}
	if agent.MaxIterations > 100 {
		return fmt.Errorf("%w: maxIterations cannot exceed 100", ErrInvalidConfig)
	}

	return nil
}

// isValidAgentName 校验 name 格式
// 规则：
// - 只能包含字母、数字、连字符、下划线
// - 不能以连字符开头或结尾
// - 不能包含连续连字符
// - 长度 1-64
func isValidAgentName(name string) bool {
	if name == "" {
		return false
	}

	// 不能以连字符开头或结尾
	if name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}

	// 不能包含连续连字符
	if strings.Contains(name, "--") {
		return false
	}

	// 只能包含字母、数字、连字符、下划线
	// 支持大写字母以兼容现有 Agent 名称（如 RepoInitializer）
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validPattern.MatchString(name)
}
