package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSkillIntegration 完整的 Skill 使用流程测试
func TestSkillIntegration(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "skills-integration-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. 创建测试 Skill: code-review
	createCodeReviewSkill(t, tmpDir)

	// 2. 创建测试 Skill: security-check
	createSecurityCheckSkill(t, tmpDir)

	// 3. 初始化 Manager
	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false, // 测试中禁用热加载
		ReloadInterval: 5 * time.Second,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	// 4. 验证 Skills 已加载
	skills := manager.Registry.List()
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}

	// 验证 code-review skill
	crSkill, err := manager.Registry.Get("code-review")
	if err != nil {
		t.Errorf("Failed to get code-review skill: %v", err)
	}
	if !crSkill.HasReferences {
		t.Error("code-review skill should have references")
	}

	// 5. 测试匹配：代码审查任务
	t.Run("MatchCodeReviewTask", func(t *testing.T) {
		task := Task{
			Type:        "code-review",
			Description: "Please review this Go code for potential issues",
			RepoType:    "go",
			Tags:        []string{"quality"},
		}

		matches, err := manager.Matcher.Match(task)
		if err != nil {
			t.Fatalf("Match failed: %v", err)
		}

		if len(matches) == 0 {
			t.Fatal("Expected at least one match")
		}

		// 第一个应该是 code-review
		if matches[0].Skill.Name != "code-review" {
			t.Errorf("Expected first match to be 'code-review', got '%s'", matches[0].Skill.Name)
		}

		t.Logf("Matched %d skills:", len(matches))
		for _, m := range matches {
			t.Logf("  - %s: %.0f%% (%s)", m.Skill.Name, m.Score*100, m.Reason)
		}
	})

	// 6. 测试匹配：安全审查任务
	t.Run("MatchSecurityTask", func(t *testing.T) {
		task := Task{
			Type:        "security",
			Description: "Check for SQL injection and XSS vulnerabilities",
			RepoType:    "python",
		}

		matches, err := manager.Matcher.Match(task)
		if err != nil {
			t.Fatalf("Match failed: %v", err)
		}

		if len(matches) == 0 {
			t.Fatal("Expected at least one match")
		}

		// 第一个应该是 security-check
		if matches[0].Skill.Name != "security-check" {
			t.Errorf("Expected first match to be 'security-check', got '%s'", matches[0].Skill.Name)
		}
	})

	// 7. 测试 Prompt 注入
	t.Run("InjectToPrompt", func(t *testing.T) {
		systemPrompt := "You are a helpful assistant."
		task := Task{
			Type:        "code-review",
			Description: "Review Go code",
			RepoType:    "go",
		}

		newPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
		if err != nil {
			t.Fatalf("MatchAndInject failed: %v", err)
		}

		if len(matches) == 0 {
			t.Fatal("Expected matches")
		}

		// 验证 Prompt 包含 Skill 内容
		if !strings.Contains(newPrompt, "专业技能指导") {
			t.Error("Prompt should contain '专业技能指导'")
		}

		if !strings.Contains(newPrompt, "code-review") {
			t.Error("Prompt should contain 'code-review'")
		}

		if !strings.Contains(newPrompt, "代码审查指南") {
			t.Error("Prompt should contain skill instructions")
		}

		t.Logf("Injected prompt length: %d", len(newPrompt))
	})

	// 8. 测试获取 Skill 内容
	t.Run("GetSkillContent", func(t *testing.T) {
		skill, body, err := manager.GetSkillContent("code-review")
		if err != nil {
			t.Fatalf("GetSkillContent failed: %v", err)
		}

		if skill.Name != "code-review" {
			t.Errorf("Expected skill name 'code-review', got '%s'", skill.Name)
		}

		if body == "" {
			t.Error("Skill body should not be empty")
		}

		if !strings.Contains(body, "代码审查指南") {
			t.Error("Body should contain '代码审查指南'")
		}
	})

	// 9. 测试加载 references
	t.Run("LoadReferences", func(t *testing.T) {
		skill, err := manager.Registry.Get("code-review")
		if err != nil {
			t.Fatal(err)
		}

		refs, err := manager.Loader.LoadReferences(skill)
		if err != nil {
			t.Fatalf("LoadReferences failed: %v", err)
		}

		if len(refs) != 1 {
			t.Errorf("Expected 1 reference, got %d", len(refs))
		}

		if content, ok := refs["CHECKLIST.md"]; !ok {
			t.Error("Expected CHECKLIST.md reference")
		} else if !strings.Contains(content, "代码审查检查清单") {
			t.Error("Reference content should contain '代码审查检查清单'")
		}
	})

	// 10. 测试禁用/启用 Skill
	t.Run("EnableDisableSkill", func(t *testing.T) {
		// 禁用 code-review
		if err := manager.Registry.Disable("code-review"); err != nil {
			t.Fatalf("Disable failed: %v", err)
		}

		// 验证禁用后匹配不到
		task := Task{
			Type:        "code-review",
			Description: "Review code",
		}

		matches, _ := manager.Matcher.Match(task)
		for _, m := range matches {
			if m.Skill.Name == "code-review" {
				t.Error("Disabled skill should not appear in matches")
			}
		}

		// 重新启用
		if err := manager.Registry.Enable("code-review"); err != nil {
			t.Fatalf("Enable failed: %v", err)
		}

		if !manager.Registry.IsEnabled("code-review") {
			t.Error("Skill should be enabled")
		}
	})
}

// TestSkillMatchingScenarios 各种匹配场景测试
func TestSkillMatchingScenarios(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-match-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个 Skills
	skills := []struct {
		name        string
		description string
	}{
		{
			name:        "go-microservice",
			description: "Analyze Go microservice architecture, service boundaries, and inter-service communication",
		},
		{
			name:        "react-frontend",
			description: "Review React components, hooks usage, and state management patterns",
		},
		{
			name:        "database-design",
			description: "Optimize database schema, indexes, and query performance",
		},
		{
			name:        "api-design",
			description: "Design RESTful and GraphQL APIs with proper versioning and documentation",
		},
	}

	for _, s := range skills {
		createSimpleSkill(t, tmpDir, s.name, s.description)
	}

	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false,
		ReloadInterval: 5 * time.Second,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	testCases := []struct {
		name         string
		task         Task
		expectFirst  string
		minMatches   int
	}{
		{
			name:         "Go microservice match",
			task:         Task{Type: "architecture", Description: "Analyze our Go microservice structure", RepoType: "go"},
			expectFirst:  "go-microservice",
			minMatches:   2,
		},
		{
			name:         "React frontend match",
			task:         Task{Type: "frontend", Description: "Review React hooks usage", RepoType: "javascript"},
			expectFirst:  "react-frontend",
			minMatches:   1,
		},
		{
			name:         "Database optimization match",
			task:         Task{Type: "performance", Description: "Optimize slow database queries"},
			expectFirst:  "database-design",
			minMatches:   1,
		},
		{
			name:         "API design match",
			task:         Task{Type: "design", Description: "Design REST API for new feature"},
			expectFirst:  "api-design",
			minMatches:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches, err := manager.Matcher.Match(tc.task)
			if err != nil {
				t.Fatalf("Match failed: %v", err)
			}

			if len(matches) < tc.minMatches {
				t.Errorf("Expected at least %d matches, got %d", tc.minMatches, len(matches))
			}

			if len(matches) > 0 && matches[0].Skill.Name != tc.expectFirst {
				t.Errorf("Expected first match to be '%s', got '%s'", tc.expectFirst, matches[0].Skill.Name)
			}

			t.Logf("Task: %s", tc.task.Description)
			for i, m := range matches {
				if i >= 3 { // 只显示前3个
					break
				}
				t.Logf("  %d. %s (%.0f%%)", i+1, m.Skill.Name, m.Score*100)
			}
		})
	}
}

// TestSkillReload 测试 Skill 重新加载
func TestSkillReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-reload-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建初始 Skill
	skillDir := createSimpleSkill(t, tmpDir, "reload-test", "Initial description")

	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false,
		ReloadInterval: 5 * time.Second,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	// 验证初始状态
	skill, _ := manager.Registry.Get("reload-test")
	if skill.Description != "Initial description" {
		t.Errorf("Initial description mismatch: %s", skill.Description)
	}

	// 修改 SKILL.md
	newContent := `---
name: reload-test
description: Updated description after reload
---

# Updated Content

This is the updated content.
`
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMDPath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 重新加载
	reloadedSkill, err := manager.Loader.Reload("reload-test")
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if reloadedSkill.Description != "Updated description after reload" {
		t.Errorf("Updated description mismatch: %s", reloadedSkill.Description)
	}

	// 验证 body 也更新了
	body, _ := manager.Loader.GetBody("reload-test")
	if !strings.Contains(body, "Updated Content") {
		t.Error("Body should contain 'Updated Content'")
	}
}

// 辅助函数：创建代码审查 Skill
func createCodeReviewSkill(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "code-review")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建 SKILL.md
	skillMD := `---
name: code-review
description: Review code for quality, style issues, and potential bugs. Use when asked to review code, check for issues, or improve code quality.
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
---

# 代码审查指南

## 审查步骤

1. **整体理解**
   - 理解代码的功能和目的
   - 检查是否符合需求

2. **代码质量**
   - 可读性：命名是否清晰，注释是否充分
   - 简洁性：是否有冗余代码
   - 一致性：是否遵循项目规范

3. **潜在问题**
   - 空指针/空值检查
   - 资源泄漏（文件、连接等）
   - 并发安全问题
   - 异常处理

4. **性能考虑**
   - 算法复杂度
   - 不必要的循环或递归
   - 内存使用

## 输出格式

对每个问题提供：
- 位置（文件:行号）
- 问题描述
- 严重程度（高/中/低）
- 改进建议
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建 references 目录
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建检查清单
	checklist := `# 代码审查检查清单

## 功能性
- [ ] 代码实现了预期功能
- [ ] 边界条件处理正确
- [ ] 错误处理完善

## 可读性
- [ ] 命名清晰有意义
- [ ] 函数长度适中
- [ ] 注释必要且准确

## 安全性
- [ ] 输入验证
- [ ] 防止注入攻击
- [ ] 敏感信息保护
`
	if err := os.WriteFile(filepath.Join(refsDir, "CHECKLIST.md"), []byte(checklist), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}

// 辅助函数：创建安全检查 Skill
func createSecurityCheckSkill(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "security-check")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: security-check
description: Check code for security vulnerabilities including SQL injection, XSS, CSRF, and sensitive data exposure. Use when reviewing security or handling user input.
license: MIT
metadata:
  author: security-team
  version: "1.0"
---

# 安全审查指南

## 常见漏洞检查

### 1. SQL 注入
- 检查字符串拼接 SQL
- 确认使用参数化查询

### 2. XSS（跨站脚本）
- 检查用户输入输出
- 确认 HTML 转义

### 3. CSRF（跨站请求伪造）
- 检查敏感操作保护
- 确认 Token 验证

### 4. 敏感信息
- 检查硬编码密码
- 确认日志脱敏
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}

// 辅助函数：创建简单 Skill
func createSimpleSkill(t *testing.T, parentDir, name, description string) string {
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
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}
