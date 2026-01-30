package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser Skill 解析器
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

// Parse 完整解析 Skill 目录
func (p *Parser) Parse(skillPath string) (*Skill, string, error) {
	skillPath = filepath.Clean(skillPath)
	skillMDPath := filepath.Join(skillPath, "SKILL.md")

	// 检查 SKILL.md 是否存在
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("%w: %s", ErrSkillMDNotFound, skillMDPath)
	}

	// 读取文件内容
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// 解析 frontmatter 和 body
	skill, body, err := p.parseSkillMD(string(content))
	if err != nil {
		return nil, "", err
	}

	// 设置路径信息
	skill.Path = skillPath
	skill.SkillMDPath = skillMDPath
	skill.LoadedAt = Now()

	// 检查资源目录
	skill.HasScripts = p.dirExists(filepath.Join(skillPath, "scripts"))
	skill.HasReferences = p.dirExists(filepath.Join(skillPath, "references"))
	skill.HasAssets = p.dirExists(filepath.Join(skillPath, "assets"))

	// 校验
	if err := p.Validate(skill); err != nil {
		return nil, "", err
	}

	return skill, body, nil
}

// ParseMetadata 仅解析元数据（快速）
func (p *Parser) ParseMetadata(skillPath string) (*Skill, error) {
	skillPath = filepath.Clean(skillPath)
	skillMDPath := filepath.Join(skillPath, "SKILL.md")

	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, err
	}

	skill, _, err := p.parseSkillMD(string(content))
	if err != nil {
		return nil, err
	}

	skill.Path = skillPath
	skill.SkillMDPath = skillMDPath
	skill.LoadedAt = Now()

	return skill, nil
}

// parseSkillMD 解析 SKILL.md 内容
func (p *Parser) parseSkillMD(content string) (*Skill, string, error) {
	skill := &Skill{}

	// 标准化换行符
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// 检查 frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", fmt.Errorf("%w: SKILL.md must start with YAML frontmatter", ErrInvalidFrontmatter)
	}

	// 找到 frontmatter 结束位置
	endIdx := strings.Index(content[3:], "\n---")
	if endIdx == -1 {
		return nil, "", fmt.Errorf("%w: YAML frontmatter not properly closed", ErrInvalidFrontmatter)
	}
	endIdx += 3 // 加上前面的 "---"

	// 提取 YAML
	yamlContent := content[3:endIdx]
	body := strings.TrimSpace(content[endIdx+4:])

	// 解析 YAML
	if err := yaml.Unmarshal([]byte(yamlContent), skill); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	return skill, body, nil
}

// Validate 校验 Skill
func (p *Parser) Validate(skill *Skill) error {
	// 校验 name
	if skill.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidName)
	}
	if len(skill.Name) > p.maxNameLen {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidName, p.maxNameLen)
	}
	if !isValidSkillName(skill.Name) {
		return fmt.Errorf("%w: name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with hyphen", ErrInvalidName)
	}

	// 校验 description
	if skill.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidDescription)
	}
	if len(skill.Description) > p.maxDescriptionLen {
		return fmt.Errorf("%w: description exceeds %d characters", ErrInvalidDescription, p.maxDescriptionLen)
	}

	return nil
}

// dirExists 检查目录是否存在
func (p *Parser) dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// isValidSkillName 校验 name 格式
// 规则：
// - 只能包含小写字母、数字、连字符
// - 不能以连字符开头或结尾
// - 不能包含连续连字符
// - 长度 1-64
func isValidSkillName(name string) bool {
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
