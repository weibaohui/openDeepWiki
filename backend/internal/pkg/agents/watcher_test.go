package agents

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileWatcher(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 收集事件
	var events []FileEvent
	var mu sync.Mutex

	callback := func(event FileEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	// 创建 watcher
	watcher := NewFileWatcher(tmpDir, 50*time.Millisecond, callback)

	// 启动 watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() unexpected error = %v", err)
	}

	// 等待初始扫描
	time.Sleep(100 * time.Millisecond)

	// 创建文件
	testFile := filepath.Join(tmpDir, "test-agent.yaml")
	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 等待事件
	time.Sleep(100 * time.Millisecond)

	// 修改文件
	newContent := `name: test-agent
version: v2
description: Updated agent
systemPrompt: You are an updated test agent.
`
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// 等待事件
	time.Sleep(100 * time.Millisecond)

	// 停止 watcher
	watcher.Stop()

	// 验证事件
	mu.Lock()
	defer mu.Unlock()

	// 应该至少收到 create 和 modify 事件
	if len(events) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}

	// 查找 create 事件
	var hasCreate bool
	var hasModify bool
	for _, e := range events {
		if e.Type == "create" && e.Path == testFile {
			hasCreate = true
		}
		if e.Type == "modify" && e.Path == testFile {
			hasModify = true
		}
	}

	if !hasCreate {
		t.Error("Expected create event")
	}

	if !hasModify {
		t.Error("Expected modify event")
	}
}

func TestFileWatcher_Delete(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 先创建文件
	testFile := filepath.Join(tmpDir, "test-agent.yaml")
	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 收集事件
	var events []FileEvent
	var mu sync.Mutex

	callback := func(event FileEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	// 创建并启动 watcher
	watcher := NewFileWatcher(tmpDir, 50*time.Millisecond, callback)
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() unexpected error = %v", err)
	}

	// 等待初始扫描
	time.Sleep(100 * time.Millisecond)

	// 删除文件
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	// 等待事件
	time.Sleep(100 * time.Millisecond)

	// 停止 watcher
	watcher.Stop()

	// 验证事件
	mu.Lock()
	defer mu.Unlock()

	// 查找 delete 事件
	var hasDelete bool
	for _, e := range events {
		if e.Type == "delete" && e.Path == testFile {
			hasDelete = true
			break
		}
	}

	if !hasDelete {
		t.Error("Expected delete event")
	}
}

func TestFileWatcher_IgnoreNonConfigFiles(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 收集事件
	var events []FileEvent
	var mu sync.Mutex

	callback := func(event FileEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	// 创建 watcher
	watcher := NewFileWatcher(tmpDir, 50*time.Millisecond, callback)

	// 启动 watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() unexpected error = %v", err)
	}

	// 等待初始扫描
	time.Sleep(100 * time.Millisecond)

	// 创建非配置文件
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 创建子目录
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// 等待
	time.Sleep(100 * time.Millisecond)

	// 停止 watcher
	watcher.Stop()

	// 验证事件（应该没有 txt 文件的事件）
	mu.Lock()
	defer mu.Unlock()

	for _, e := range events {
		if e.Path == testFile {
			t.Error("Should not receive events for .txt files")
		}
	}
}

func TestFileWatcher_MultipleExtensions(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 收集事件
	var events []FileEvent
	var mu sync.Mutex

	callback := func(event FileEvent) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}

	// 创建 watcher
	watcher := NewFileWatcher(tmpDir, 50*time.Millisecond, callback)

	// 启动 watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() unexpected error = %v", err)
	}

	// 等待初始扫描
	time.Sleep(100 * time.Millisecond)

	// 创建不同扩展名的文件
	files := []string{
		"agent1.yaml",
		"agent2.yml",
		"agent3.JSON", // 大写扩展名
	}

	for _, filename := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// 等待事件
	time.Sleep(100 * time.Millisecond)

	// 停止 watcher
	watcher.Stop()

	// 验证事件
	mu.Lock()
	defer mu.Unlock()

	// 应该收到所有 3 个 create 事件
	var createCount int
	for _, e := range events {
		if e.Type == "create" {
			createCount++
		}
	}

	if createCount != 3 {
		t.Errorf("Expected 3 create events, got %d", createCount)
	}
}

func TestFileWatcher_Stop(t *testing.T) {
	tmpDir := t.TempDir()

	watcher := NewFileWatcher(tmpDir, time.Second, func(event FileEvent) {})

	// 启动
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() unexpected error = %v", err)
	}

	// 停止
	watcher.Stop()

	// 再次停止不应 panic
	watcher.Stop()
}
