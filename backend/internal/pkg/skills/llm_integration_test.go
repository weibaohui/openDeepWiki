package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// TestSkillWithLLMFlow 测试 Skills 与 LLM 的完整集成流程
func TestSkillWithLLMFlow(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "skills-llm-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. 创建测试 Skill
	createGoAnalysisSkillForLLM(t, tmpDir)

	// 2. 初始化 Manager
	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false,
		ReloadInterval: 5 * time.Second,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Stop()

	// 3. 测试带有 Skills 的 LLM 对话流程
	t.Run("LLMConversationWithSkill", func(t *testing.T) {
		// 原始 System Prompt
		systemPrompt := "You are a code analysis assistant. \nYour task is to analyze code repositories and provide insights.\nBe concise and focus on architectural patterns."

		// 任务定义
		task := Task{
			Type:        "architecture",
			Description: "分析这个 Go 项目的架构和模块设计",
			RepoType:    "go",
			Tags:        []string{"microservice", "clean-architecture"},
		}

		// 4. 匹配 Skills 并注入 Prompt
		enhancedPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
		if err != nil {
			t.Fatalf("MatchAndInject failed: %v", err)
		}

		if len(matches) == 0 {
			t.Fatal("Expected at least one skill match")
		}

		// 验证增强后的 Prompt
		if !strings.Contains(enhancedPrompt, "专业技能指导") {
			t.Error("Enhanced prompt should contain '专业技能指导'")
		}

		if !strings.Contains(enhancedPrompt, "go-analysis") {
			t.Error("Enhanced prompt should contain skill name")
		}

		// 验证 Prompt 结构
		if !strings.Contains(enhancedPrompt, systemPrompt) {
			t.Error("Enhanced prompt should include original system prompt")
		}

		t.Logf("Original prompt length: %d", len(systemPrompt))
		t.Logf("Enhanced prompt length: %d", len(enhancedPrompt))
		t.Logf("Matched skills: %d", len(matches))
		for _, m := range matches {
			t.Logf("  - %s (%.0f%%): %s", m.Skill.Name, m.Score*100, m.Reason)
		}

		// 5. 模拟构建 LLM 请求
		messages := []llm.ChatMessage{
			{Role: "system", Content: enhancedPrompt},
			{Role: "user", Content: "请分析这个 Go 项目的架构：github.com/example/myapp"},
		}

		// 验证消息格式
		if messages[0].Role != "system" {
			t.Error("First message should be system")
		}

		if !strings.Contains(messages[0].Content, "Go 项目分析指南") {
			t.Error("System message should contain skill instructions")
		}

		t.Logf("LLM Messages:")
		t.Logf("  System: %d chars", len(messages[0].Content))
		t.Logf("  User: %s", messages[1].Content)
	})

	// 4. 测试多个 Skills 组合
	t.Run("MultipleSkillsCombination", func(t *testing.T) {
		// 创建第二个 Skill
		createDocGenSkillForLLM(t, tmpDir)

		// 重新加载
		_, err := manager.Loader.LoadFromDir(tmpDir)
		if err != nil {
			t.Logf("Reload warning: %v", err)
		}

		systemPrompt := "You are a technical assistant."
		task := Task{
			Type:        "documentation",
			Description: "为 Go 项目生成架构文档",
			RepoType:    "go",
		}

		newPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
		if err != nil {
			t.Fatalf("MatchAndInject failed: %v", err)
		}

		// 验证两个 skills 都匹配了
		hasGoAnalysis := false
		hasDocGen := false
		for _, m := range matches {
			if m.Skill.Name == "go-analysis" {
				hasGoAnalysis = true
			}
			if m.Skill.Name == "doc-generation" {
				hasDocGen = true
			}
		}

		if !hasGoAnalysis {
			t.Error("Should match go-analysis skill")
		}
		if !hasDocGen {
			t.Error("Should match doc-generation skill")
		}

		// 验证两个 skill 的内容都在 prompt 中
		if !strings.Contains(newPrompt, "Go 项目分析指南") {
			t.Error("Prompt should contain go-analysis instructions")
		}
		if !strings.Contains(newPrompt, "文档生成指南") {
			t.Error("Prompt should contain doc-generation instructions")
		}

		t.Logf("Combined %d skills into prompt of %d chars", len(matches), len(newPrompt))
	})

	// 5. 测试 Skill 匹配但未使用（低匹配度）
	t.Run("LowRelevanceTask", func(t *testing.T) {
		systemPrompt := "You are a helpful assistant."
		task := Task{
			Type:        "frontend",
			Description: "Create a React component for user login",
			RepoType:    "javascript",
		}

		newPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
		if err != nil {
			t.Fatalf("MatchAndInject failed: %v", err)
		}

		// 对于不相关的任务，可能匹配度很低或没有匹配
		t.Logf("Task: %s", task.Description)
		t.Logf("Matched skills: %d", len(matches))

		// Prompt 应该基本保持不变（或只有轻微增强）
		if len(matches) == 0 {
			if newPrompt != systemPrompt {
				t.Error("With no matches, prompt should remain unchanged")
			}
		}
	})
}

// TestSkillWithLLMToolCalls 测试 Skills 配合 LLM Tool 使用
func TestSkillWithLLMToolCalls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-llm-tools-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建带有工具使用指导的 Skill
	createSkillWithTools(t, tmpDir)

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

	t.Run("SkillGuidesToolUsage", func(t *testing.T) {
		systemPrompt := "You are a code analysis assistant."
		task := Task{
			Type:        "analysis",
			Description: "分析项目并读取关键文件",
			RepoType:    "go",
		}

		enhancedPrompt, matches, err := manager.MatchAndInject(systemPrompt, task)
		if err != nil {
			t.Fatal(err)
		}

		if len(matches) == 0 {
			t.Fatal("Expected skill match")
		}

		// 验证 Prompt 包含工具使用指导
		if !strings.Contains(enhancedPrompt, "ReadFile") {
			t.Error("Prompt should mention ReadFile tool")
		}
		if !strings.Contains(enhancedPrompt, "SearchFiles") {
			t.Error("Prompt should mention SearchFiles tool")
		}

		// 6. 构建 LLM 请求（带 Tools）
		tools := []llm.Tool{
			llm.ReadFileTool(),
			llm.SearchFilesTool(),
			llm.SearchTextTool(),
		}

		request := llm.ChatRequest{
			Model: "gpt-4",
			Messages: []llm.ChatMessage{
				{Role: "system", Content: enhancedPrompt},
				{Role: "user", Content: "分析这个 Go 项目: /path/to/project"},
			},
			Tools:      tools,
			ToolChoice: "auto",
		}

		// 验证请求结构
		if len(request.Messages) != 2 {
			t.Error("Should have 2 messages")
		}

		if len(request.Tools) != 3 {
			t.Error("Should have 3 tools")
		}

		// 验证 Skill 指导在 system message 中
		if !strings.Contains(request.Messages[0].Content, "代码分析工具使用指南") {
			t.Error("System message should contain tool usage guidance from skill")
		}

		t.Logf("ChatRequest: Model=%s, Tools=%d", request.Model, len(request.Tools))
	})
}

// TestSkillPromptBuilding 测试不同方式的 Prompt 构建
func TestSkillPromptBuilding(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-prompt-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	createGoAnalysisSkillForLLM(t, tmpDir)

	config := &Config{
		Dir:        tmpDir,
		AutoReload: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	t.Run("DirectSkillInjection", func(t *testing.T) {
		// 直接使用 Injector 构建 context
		task := Task{
			Description: "分析 Go 项目",
			RepoType:    "go",
		}

		matches, err := manager.Matcher.Match(task)
		if err != nil {
			t.Fatal(err)
		}

		// 构建 skill context
		skillContext, err := manager.Injector.BuildSkillContext(matches)
		if err != nil {
			t.Fatal(err)
		}

		// 手动组装 prompt
		fullPrompt := "You are an expert.\n\n" + skillContext + "\n\nNow analyze the code."

		if !strings.Contains(fullPrompt, "专业技能指导") {
			t.Error("Should contain skill context")
		}

		t.Logf("Custom prompt length: %d", len(fullPrompt))
	})

	t.Run("SingleSkillContext", func(t *testing.T) {
		skill, err := manager.Registry.Get("go-analysis")
		if err != nil {
			t.Fatal(err)
		}

		context, err := manager.Injector.BuildSingleSkillContext(skill)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(context, "技能: go-analysis") {
			t.Error("Should contain skill header")
		}

		t.Logf("Single skill context length: %d", len(context))
	})
}

// TestSkillMetadataForLLM 测试 Skill 元数据在 LLM 上下文中的使用
func TestSkillMetadataForLLM(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skills-metadata-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建带有丰富元数据的 Skill
	createSkillWithMetadata(t, tmpDir)

	config := &Config{
		Dir:        tmpDir,
		AutoReload: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	t.Run("UseMetadataInPrompt", func(t *testing.T) {
		skill, err := manager.Registry.Get("advanced-go")
		if err != nil {
			t.Fatal(err)
		}

		// 验证元数据可用于构建提示
		var sb strings.Builder
		sb.WriteString("You are using skill: " + skill.Name + "\n")
		sb.WriteString("Description: " + skill.Description + "\n")
		if skill.Compatibility != "" {
			sb.WriteString("Requirements: " + skill.Compatibility + "\n")
		}
		if len(skill.Metadata) > 0 {
			sb.WriteString("Metadata:\n")
			for k, v := range skill.Metadata {
				sb.WriteString("  - " + k + ": " + v + "\n")
			}
		}

		prompt := sb.String()

		if !strings.Contains(prompt, "advanced-go") {
			t.Error("Prompt should contain skill name")
		}
		if !strings.Contains(prompt, "Requires Go 1.21+") {
			t.Error("Prompt should contain compatibility info")
		}
		if !strings.Contains(prompt, "expert") {
			t.Error("Prompt should contain metadata")
		}

		t.Logf("Metadata-enhanced prompt:\n%s", prompt)
	})
}

// 辅助函数：创建用于 LLM 测试的 Go 分析 Skill
func createGoAnalysisSkillForLLM(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "go-analysis")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: go-analysis
description: Analyze Go projects to identify architecture patterns, module dependencies, API endpoints, and code organization. Use when working with Go repositories or when the user asks about Go project structure, module design, or architectural analysis.
license: MIT
compatibility: Requires Go 1.18+
metadata:
  author: openDeepWiki
  version: "1.0"
  category: code-analysis
  priority: high
---

# Go 项目分析指南

## 分析步骤

1. **识别项目结构**
   - 查找 go.mod 了解模块路径和依赖
   - 分析目录结构，识别主要包
   - 找出入口点（main 包）

2. **分析架构模式**
   - 分层架构（handler -> service -> repository）
   - 接口定义和实现分离
   - 依赖注入的使用

3. **核心组件识别**
   - HTTP/gRPC handlers
   - Service/UseCase 层
   - Repository/DAO 层

4. **依赖关系分析**
   - 内部包间的 import 关系
   - 第三方库的使用场景

## 输出要求

生成以下文档：
- overview.md: 项目概述
- architecture.md: 架构分析
- api.md: API 文档

## 注意事项

- 关注 internal/ 包的封装性
- 注意接口定义的抽象程度
- 检查错误处理模式
- 识别并发和通道的使用
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}

// 辅助函数：创建文档生成 Skill
func createDocGenSkillForLLM(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "doc-generation")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: doc-generation
description: Generate comprehensive technical documentation for software projects including architecture docs, API documentation, and developer guides. Use when creating or updating project documentation.
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: documentation
---

# 文档生成指南

## 文档类型

### 1. 项目概览 (overview.md)
- 项目简介
- 技术栈清单
- 主要功能特性
- 快速开始指南

### 2. 架构文档 (architecture.md)
- 系统架构图（使用 Mermaid）
- 模块划分说明
- 数据流图
- 技术选型理由

### 3. API 文档 (api.md)
- 认证方式说明
- 错误码定义
- 接口列表

## 规范要求

- 使用 Markdown 格式
- 代码块指定语言
- 使用 Mermaid 绘制图表
- 保持客观中立的语气
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}

// 辅助函数：创建带有工具指导的 Skill
func createSkillWithTools(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "analysis-with-tools")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: analysis-with-tools
description: Guide for analyzing Go projects using file reading and search tools. Use when analyzing Go projects or reading source code files.
---

# 代码分析工具使用指南

## 推荐工具使用顺序

1. **SearchFiles** - 查找项目结构
   - 先查找所有 .go 文件了解项目规模
   - 查找 go.mod, main.go 等关键文件

2. **ReadFile** - 读取关键文件
   - 读取 go.mod 了解依赖
   - 读取 main 函数了解入口
   - 读取核心接口定义

3. **SearchText** - 搜索特定模式
   - 搜索接口定义
   - 搜索结构体定义
   - 搜索函数签名

## 分析策略

- 从入口点开始，逐步深入
- 先理解整体架构，再关注细节
- 记录关键发现，用于生成文档
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}

// 辅助函数：创建带有丰富元数据的 Skill
func createSkillWithMetadata(t *testing.T, parentDir string) string {
	skillDir := filepath.Join(parentDir, "advanced-go")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: advanced-go
description: Advanced Go analysis with generics, fuzzing, and workspace support. Use for modern Go projects using latest features.
license: Apache-2.0
compatibility: Requires Go 1.21+
metadata:
  author: go-expert
  version: "2.0"
  level: advanced
  tags: generics fuzzing workspaces
---

# 高级 Go 分析

## 新特性支持

- Generics 分析
- Fuzzing 测试识别
- Workspace 模式
- 标准库新包
`

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	return skillDir
}
