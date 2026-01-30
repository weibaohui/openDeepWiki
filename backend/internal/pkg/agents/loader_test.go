package agents

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoader_LoadFromDir(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试配置文件
	configs := map[string]string{
		"agent1.yaml": `name: agent-1
version: v1
description: Test agent 1
systemPrompt: You are agent 1.
`,
		"agent2.yml": `name: agent-2
version: v1
description: Test agent 2
systemPrompt: You are agent 2.
`,
		"agent3.json": `{
  "name": "agent-3",
  "version": "v1",
  "description": "Test agent 3",
  "systemPrompt": "You are agent 3."
}`,
		"invalid.txt": `this should be ignored`,
	}

	for filename, content := range configs {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// 创建 loader
	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 加载目录
	results, err := loader.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error = %v", err)
	}

	// 验证结果（应该加载 3 个 agents，跳过 .txt 文件）
	if len(results) != 3 {
		t.Errorf("LoadFromDir() loaded %d agents, expected 3", len(results))
	}

	// 验证所有 agents 都被注册
	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, name := range agents {
		if !reg.Exists(name) {
			t.Errorf("Agent %s should be registered", name)
		}
	}

	// 验证 action 都是 created
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("LoadFromDir() result error = %v", result.Error)
		}
		if result.Action != "created" {
			t.Errorf("LoadFromDir() action = %v, expected created", result.Action)
		}
	}
}

func TestLoader_LoadFromDir_NonExistent(t *testing.T) {
	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 加载不存在的目录
	results, err := loader.LoadFromDir("/non/existent/dir")
	if err != nil {
		t.Errorf("LoadFromDir() unexpected error = %v", err)
	}

	if results != nil {
		t.Error("LoadFromDir() should return nil for non-existent directory")
	}
}

func TestLoader_LoadFromDir_Empty(t *testing.T) {
	// 创建空临时目录
	tmpDir := t.TempDir()

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	results, err := loader.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("LoadFromDir() loaded %d agents, expected 0", len(results))
	}
}

func TestLoader_LoadFromPath(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-agent.yaml")

	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 加载单个文件
	agent, err := loader.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() unexpected error = %v", err)
	}

	if agent.Name != "test-agent" {
		t.Errorf("LoadFromPath() agent.Name = %v, want test-agent", agent.Name)
	}

	// 验证已注册
	if !reg.Exists("test-agent") {
		t.Error("Agent should be registered after LoadFromPath")
	}
}

func TestLoader_LoadFromPath_Invalid(t *testing.T) {
	// 创建临时文件（无效配置）
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-agent.yaml")

	content := `name: ""
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 加载无效配置
	_, err := loader.LoadFromPath(configPath)
	if err == nil {
		t.Error("LoadFromPath() expected error for invalid config, got nil")
	}
}

func TestLoader_Reload(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-agent.yaml")

	content := `name: test-agent
version: v1
description: Original description
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 首次加载
	agent, err := loader.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() unexpected error = %v", err)
	}

	originalLoadedAt := agent.LoadedAt

	// 等待一小段时间确保时间不同
	time.Sleep(10 * time.Millisecond)

	// 修改文件
	newContent := `name: test-agent
version: v2
description: Updated description
systemPrompt: You are an updated test agent.
`
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// 重新加载
	reloadedAgent, err := loader.Reload("test-agent")
	if err != nil {
		t.Fatalf("Reload() unexpected error = %v", err)
	}

	// 验证更新
	if reloadedAgent.Version != "v2" {
		t.Errorf("Reload() Version = %v, want v2", reloadedAgent.Version)
	}

	if reloadedAgent.Description != "Updated description" {
		t.Errorf("Reload() Description = %v, want 'Updated description'", reloadedAgent.Description)
	}

	if !reloadedAgent.LoadedAt.After(originalLoadedAt) {
		t.Error("Reload() LoadedAt should be updated")
	}
}

func TestLoader_Reload_NonExistent(t *testing.T) {
	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	_, err := loader.Reload("non-existent")
	if err == nil {
		t.Error("Reload() expected error for non-existent agent, got nil")
	}
}

func TestLoader_Unload(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-agent.yaml")

	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 加载
	if _, err := loader.LoadFromPath(configPath); err != nil {
		t.Fatalf("LoadFromPath() unexpected error = %v", err)
	}

	// 验证存在
	if !reg.Exists("test-agent") {
		t.Fatal("Agent should exist after loading")
	}

	// 卸载
	if err := loader.Unload("test-agent"); err != nil {
		t.Errorf("Unload() unexpected error = %v", err)
	}

	// 验证不存在
	if reg.Exists("test-agent") {
		t.Error("Agent should not exist after unloading")
	}
}

func TestLoader_Unload_NonExistent(t *testing.T) {
	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	err := loader.Unload("non-existent")
	if err == nil {
		t.Error("Unload() expected error for non-existent agent, got nil")
	}
}

func TestLoader_LoadResult(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-agent.yaml")

	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reg := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, reg)

	// 首次加载
	result, err := loader.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("LoadFromDir() returned %d results, expected 1", len(result))
	}

	if result[0].Action != "created" {
		t.Errorf("First load Action = %v, expected created", result[0].Action)
	}

	// 再次加载（应该更新）
	result, err = loader.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("LoadFromDir() returned %d results, expected 1", len(result))
	}

	if result[0].Action != "updated" {
		t.Errorf("Second load Action = %v, expected updated", result[0].Action)
	}
}
