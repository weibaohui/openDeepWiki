package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser Agent 配置解析器
type Parser struct {
	maxDescriptionLen int
	maxNameLen        int
}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{
		maxDescriptionLen: 1024,
		maxNameLen:        64,
	}
}

// Parse 解析 Agent 配置文件
func (p *Parser) Parse(configPath string) (*Agent, error) {
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
	agent := &Agent{}
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
func (p *Parser) Validate(agent *Agent) error {
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

	// 校验 version
	if agent.Version == "" {
		return fmt.Errorf("%w: version is required", ErrInvalidConfig)
	}
	if !isValidVersion(agent.Version) {
		return fmt.Errorf("%w: version must be valid semantic version (e.g., v1, v1.0, v1.0.0)", ErrInvalidConfig)
	}

	// 校验 description
	if agent.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidConfig)
	}
	if len(agent.Description) > p.maxDescriptionLen {
		return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidConfig, p.maxDescriptionLen)
	}

	// 校验 systemPrompt
	if agent.SystemPrompt == "" {
		return fmt.Errorf("%w: systemPrompt is required", ErrInvalidConfig)
	}

	// 校验 riskLevel（如果设置了）
	if agent.RuntimePolicy.RiskLevel != "" {
		validRiskLevels := map[string]bool{"read": true, "write": true, "admin": true}
		if !validRiskLevels[agent.RuntimePolicy.RiskLevel] {
			return fmt.Errorf("%w: riskLevel must be one of: read, write, admin", ErrInvalidConfig)
		}
	}

	return nil
}

// isValidAgentName 校验 name 格式
// 规则：
// - 只能包含小写字母、数字、连字符
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

	// 只能包含小写字母、数字、连字符
	validPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	return validPattern.MatchString(name)
}

// isValidVersion 校验 version 格式（简单语义化版本）
// 支持 v1, v1.0, v1.0.0 格式
func isValidVersion(version string) bool {
	if version == "" {
		return false
	}
	pattern := regexp.MustCompile(`^v\d+(\.\d+)?(\.\d+)?$`)
	return pattern.MatchString(version)
}
