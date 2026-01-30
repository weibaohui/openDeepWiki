package skills

import (
	"fmt"
	"strings"
)

// Injector Skill 注入器
type Injector struct {
	loader *Loader
}

// NewInjector 创建注入器
func NewInjector(loader *Loader) *Injector {
	return &Injector{loader: loader}
}

// InjectToPrompt 将 Skills 注入到 System Prompt
func (i *Injector) InjectToPrompt(systemPrompt string, matches []*Match) (string, error) {
	if len(matches) == 0 {
		return systemPrompt, nil
	}

	// 构建 Skills 上下文
	skillContext, err := i.BuildSkillContext(matches)
	if err != nil {
		return "", err
	}

	// 注入到 Prompt
	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n")
	sb.WriteString(skillContext)

	return sb.String(), nil
}

// BuildSkillContext 构建 Skills 上下文
func (i *Injector) BuildSkillContext(matches []*Match) (string, error) {
	var sb strings.Builder

	sb.WriteString("## 专业技能指导\n\n")
	sb.WriteString("在完成以下任务时，请参考相关技能的专业指导：\n\n")

	for idx, match := range matches {
		skill := match.Skill

		sb.WriteString(fmt.Sprintf("### 技能 %d: %s\n", idx+1, skill.Name))
		sb.WriteString(fmt.Sprintf("> **匹配度**: %.0f%%  |  **原因**: %s\n\n", match.Score*100, match.Reason))

		// 获取指令内容
		body, err := i.loader.GetBody(skill.Name)
		if err != nil {
			sb.WriteString(fmt.Sprintf("*(无法加载技能内容: %v)*\n", err))
			continue
		}

		sb.WriteString(body)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String(), nil
}

// BuildSingleSkillContext 构建单个 Skill 上下文
func (i *Injector) BuildSingleSkillContext(skill *Skill) (string, error) {
	body, err := i.loader.GetBody(skill.Name)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 技能: %s\n\n", skill.Name))
	sb.WriteString(fmt.Sprintf("> %s\n\n", skill.Description))
	sb.WriteString(body)

	return sb.String(), nil
}

// BuildMinimalContext 构建最小化上下文（仅元数据）
func (i *Injector) BuildMinimalContext(skills []*Skill) string {
	var sb strings.Builder

	sb.WriteString("## 可用技能\n\n")
	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", skill.Name, skill.Description))
	}

	return sb.String()
}
