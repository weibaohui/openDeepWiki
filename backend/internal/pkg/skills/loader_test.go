package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadFromDir(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "skills-loader-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试 Skills
	createTestSkill(t, tmpDir, "skill-one", "First test skill")
	createTestSkill(t, tmpDir, "skill-two", "Second test skill")

	// 创建非 Skill 目录（没有 SKILL.md）
	nonSkillDir := filepath.Join(tmpDir, "not-a-skill")
	if err := os.MkdirAll(nonSkillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 加载
	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	results, err := loader.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("LoadFromDir() loaded %d skills, want 2", len(results))
	}

	// 验证 Registry
	skills := registry.List()
	if len(skills) != 2 {
		t.Errorf("Registry has %d skills, want 2", len(skills))
	}
}

func TestLoader_LoadFromPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-loader-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	skillDir := createTestSkill(t, tmpDir, "test-skill", "A test skill")

	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	skill, err := loader.LoadFromPath(skillDir)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Skill.Name = %v, want test-skill", skill.Name)
	}

	// 测试获取 body
	body, err := loader.GetBody("test-skill")
	if err != nil {
		t.Fatalf("GetBody() error = %v", err)
	}

	if body == "" {
		t.Error("GetBody() returned empty body")
	}
}

func TestLoader_Unload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-loader-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	skillDir := createTestSkill(t, tmpDir, "unload-test", "Test unload")

	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	// 加载
	if _, err := loader.LoadFromPath(skillDir); err != nil {
		t.Fatal(err)
	}

	// 验证已加载
	if _, err := registry.Get("unload-test"); err != nil {
		t.Error("Skill should be registered")
	}

	// 卸载
	if err := loader.Unload("unload-test"); err != nil {
		t.Fatalf("Unload() error = %v", err)
	}

	// 验证已卸载
	if _, err := registry.Get("unload-test"); err == nil {
		t.Error("Skill should be unregistered")
	}

	// 验证 body 缓存已清除
	_, err = loader.GetBody("unload-test")
	if err == nil {
		t.Error("GetBody() should return error for unloaded skill")
	}
}

func TestLoader_Reload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-loader-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	skillDir := createTestSkill(t, tmpDir, "reload-test", "Original description")

	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)

	// 加载
	if _, err := loader.LoadFromPath(skillDir); err != nil {
		t.Fatal(err)
	}

	// 修改 SKILL.md
	skillMD := `---
name: reload-test
description: Updated description
---

# Updated Content

New instructions here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	// 重新加载
	skill, err := loader.Reload("reload-test")
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if skill.Description != "Updated description" {
		t.Errorf("Description = %v, want 'Updated description'", skill.Description)
	}

	// 验证 body 也更新了
	body, err := loader.GetBody("reload-test")
	if err != nil {
		t.Fatal(err)
	}

	if !contains(body, "Updated Content") {
		t.Error("Body should contain 'Updated Content'")
	}
}

// 辅助函数：创建测试 Skill
func createTestSkill(t *testing.T, parentDir, name, description string) string {
	skillDir := filepath.Join(parentDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: ` + name + `
description: ` + description + `
---

# ` + name + `

This is the ` + name + ` skill.

## Instructions

Test instructions here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}
