package agents

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试配置文件
	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	configPath := filepath.Join(tmpDir, "test-agent.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false, // 禁用热加载以便测试
		DefaultAgent:   "test-agent",
		Routes:         map[string]string{"test": "test-agent"},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}
	defer manager.Stop()

	// 验证组件已创建
	if manager.Registry == nil {
		t.Error("Registry should not be nil")
	}

	if manager.Parser == nil {
		t.Error("Parser should not be nil")
	}

	if manager.Loader == nil {
		t.Error("Loader should not be nil")
	}

	if manager.Router == nil {
		t.Error("Router should not be nil")
	}

	// 验证 agent 已加载
	if !manager.Registry.Exists("test-agent") {
		t.Error("test-agent should be loaded")
	}

	// 验证默认设置
	if manager.Router.GetDefault() != "test-agent" {
		t.Errorf("Default agent = %v, want test-agent", manager.Router.GetDefault())
	}

	// 验证路由规则
	agentName, exists := manager.Router.GetRoute("test")
	if !exists {
		t.Error("Route for 'test' should exist")
	}
	if agentName != "test-agent" {
		t.Errorf("Route 'test' = %v, want test-agent", agentName)
	}
}

func TestNewManager_NilConfig(t *testing.T) {
	// 使用 nil 配置
	manager, err := NewManager(nil)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}
	defer manager.Stop()

	// 验证使用默认配置
	if manager.Config.Dir == "" {
		t.Error("Config.Dir should not be empty")
	}
}

func TestNewManager_CreateDir(t *testing.T) {
	// 使用不存在的目录
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "non-existent", "agents")

	config := &Config{
		Dir:        nonExistentDir,
		AutoReload: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}
	defer manager.Stop()

	// 验证目录已创建
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Error("Directory should be created")
	}
}

func TestManager_SelectAgent(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试配置文件
	configs := map[string]string{
		"diagnose-agent.yaml": `name: diagnose-agent
version: v1
description: Diagnose agent
systemPrompt: You are a diagnose agent.
`,
		"ops-agent.yaml": `name: ops-agent
version: v1
description: Ops agent
systemPrompt: You are an ops agent.
`,
	}

	for filename, content := range configs {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	config := &Config{
		Dir:            tmpDir,
		AutoReload:     false,
		DefaultAgent:   "diagnose-agent",
		Routes:         map[string]string{"diagnose": "diagnose-agent", "ops": "ops-agent"},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}
	defer manager.Stop()

	tests := []struct {
		name    string
		ctx     RouterContext
		want    string
		wantErr bool
	}{
		{
			name: "select by explicit name",
			ctx: RouterContext{
				AgentName: "ops-agent",
			},
			want:    "ops-agent",
			wantErr: false,
		},
		{
			name: "select by entry point",
			ctx: RouterContext{
				EntryPoint: "diagnose",
			},
			want:    "diagnose-agent",
			wantErr: false,
		},
		{
			name: "select default",
			ctx: RouterContext{
				EntryPoint: "unknown",
			},
			want:    "diagnose-agent",
			wantErr: false,
		},
		{
			name: "select non-existent",
			ctx: RouterContext{
				AgentName: "non-existent",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.SelectAgent(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name != tt.want {
				t.Errorf("SelectAgent() got = %v, want %v", got.Name, tt.want)
			}
		})
	}
}

func TestManager_ReloadAll(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试配置文件
	content := `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`
	configPath := filepath.Join(tmpDir, "test-agent.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := &Config{
		Dir:        tmpDir,
		AutoReload: false,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}
	defer manager.Stop()

	// 验证初始加载
	if !manager.Registry.Exists("test-agent") {
		t.Fatal("test-agent should exist initially")
	}

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	// 更新文件
	newContent := `name: test-agent
version: v2
description: Updated agent
systemPrompt: You are an updated test agent.
`
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// 重新加载所有
	if err := manager.ReloadAll(); err != nil {
		t.Fatalf("ReloadAll() unexpected error = %v", err)
	}

	// 验证更新
	agent, err := manager.Registry.Get("test-agent")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if agent.Version != "v2" {
		t.Errorf("After reload Version = %v, want v2", agent.Version)
	}

	if agent.Description != "Updated agent" {
		t.Errorf("After reload Description = %v, want 'Updated agent'", agent.Description)
	}
}

func TestManager_Stop(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	config := &Config{
		Dir:            tmpDir,
		AutoReload:     true, // 启用热加载
		ReloadInterval: time.Second,
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() unexpected error = %v", err)
	}

	// 验证 watcher 已启动
	if manager.watcher == nil {
		t.Error("Watcher should be started")
	}

	// 停止
	manager.Stop()

	// 多次停止不应 panic
	manager.Stop()
}

func TestResolveAgentsDir(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		configDir string
		wantEnv   bool
	}{
		{
			name:      "use environment variable",
			envValue:  "/env/agents",
			configDir: "/config/agents",
			wantEnv:   true,
		},
		{
			name:      "use config dir",
			envValue:  "",
			configDir: "/config/agents",
			wantEnv:   false,
		},
		{
			name:      "use default",
			envValue:  "",
			configDir: "",
			wantEnv:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置/清除环境变量
			if tt.envValue != "" {
				os.Setenv("AGENTS_DIR", tt.envValue)
				defer os.Unsetenv("AGENTS_DIR")
			} else {
				os.Unsetenv("AGENTS_DIR")
			}

			dir, err := resolveAgentsDir(tt.configDir)
			if err != nil {
				t.Fatalf("resolveAgentsDir() unexpected error = %v", err)
			}

			if tt.wantEnv {
				if dir != tt.envValue {
					t.Errorf("resolveAgentsDir() = %v, want %v (from env)", dir, tt.envValue)
				}
			} else if tt.configDir != "" {
				if dir != tt.configDir {
					t.Errorf("resolveAgentsDir() = %v, want %v (from config)", dir, tt.configDir)
				}
			}
		})
	}
}

func TestGuessAgentNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/path/to/agent.yaml", "agent"},
		{"/path/to/agent.yml", "agent"},
		{"/path/to/agent.json", "agent"},
		{"agent.yaml", "agent"},
		{"my-test-agent.yml", "my-test-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := guessAgentNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("guessAgentNameFromPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Dir != "./agents" {
		t.Errorf("DefaultConfig().Dir = %v, want ./agents", config.Dir)
	}

	if !config.AutoReload {
		t.Error("DefaultConfig().AutoReload should be true")
	}

	if config.ReloadInterval != 5*time.Second {
		t.Errorf("DefaultConfig().ReloadInterval = %v, want 5s", config.ReloadInterval)
	}

	if config.Routes == nil {
		t.Error("DefaultConfig().Routes should not be nil")
	}
}
