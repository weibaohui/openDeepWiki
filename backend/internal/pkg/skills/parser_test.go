package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "skills-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试 Skill
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: test-skill
description: A test skill for unit testing
license: MIT
metadata:
  author: test
  version: "1.0"
---

# Test Skill

This is a test skill.

## Instructions

1. Step one
2. Step two
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	// 测试解析
	parser := NewParser()
	skill, body, err := parser.Parse(skillDir)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 验证元数据
	if skill.Name != "test-skill" {
		t.Errorf("Name = %v, want test-skill", skill.Name)
	}
	if skill.Description != "A test skill for unit testing" {
		t.Errorf("Description = %v, want 'A test skill for unit testing'", skill.Description)
	}
	if skill.License != "MIT" {
		t.Errorf("License = %v, want MIT", skill.License)
	}
	if skill.Metadata["author"] != "test" {
		t.Errorf("Metadata[author] = %v, want test", skill.Metadata["author"])
	}

	// 验证 body
	if body == "" {
		t.Error("Body should not be empty")
	}
	if !contains(body, "# Test Skill") {
		t.Error("Body should contain '# Test Skill'")
	}

	// 验证路径
	if skill.Path != skillDir {
		t.Errorf("Path = %v, want %v", skill.Path, skillDir)
	}
}

func TestParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		skill   *Skill
		wantErr bool
		errType error
	}{
		{
			name: "valid skill",
			skill: &Skill{
				Name:        "valid-skill",
				Description: "A valid skill description",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			skill: &Skill{
				Name:        "",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "name with uppercase",
			skill: &Skill{
				Name:        "Invalid-Skill",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "name starts with hyphen",
			skill: &Skill{
				Name:        "-invalid",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "name ends with hyphen",
			skill: &Skill{
				Name:        "invalid-",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "name with consecutive hyphens",
			skill: &Skill{
				Name:        "invalid--skill",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "name too long",
			skill: &Skill{
				Name:        "this-is-a-very-long-skill-name-that-exceeds-the-maximum-allowed-length-of-64-characters",
				Description: "A valid description",
			},
			wantErr: true,
			errType: ErrInvalidName,
		},
		{
			name: "empty description",
			skill: &Skill{
				Name:        "valid-name",
				Description: "",
			},
			wantErr: true,
			errType: ErrInvalidDescription,
		},
		{
			name: "description too long",
			skill: &Skill{
				Name:        "valid-name",
				Description: string(make([]byte, 1025)),
			},
			wantErr: true,
			errType: ErrInvalidDescription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidSkillName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"valid-skill", true},
		{"skill123", true},
		{"skill-123", true},
		{"a", true},
		{"", false},
		{"-skill", false},
		{"skill-", false},
		{"skill--name", false},
		{"Skill-Name", false},
		{"skill_name", false},
		{"skill.name", false},
		{"skill name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSkillName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidSkillName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestParser_ParseInvalidFrontmatter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	skillDir := filepath.Join(tmpDir, "invalid-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 缺少 frontmatter
	noFrontmatter := `# Just a markdown file`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(noFrontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, _, err = parser.Parse(skillDir)
	if err == nil {
		t.Error("Parse() should return error for missing frontmatter")
	}

	// 未闭合的 frontmatter
	unclosed := `---
name: test
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(unclosed), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err = parser.Parse(skillDir)
	if err == nil {
		t.Error("Parse() should return error for unclosed frontmatter")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
