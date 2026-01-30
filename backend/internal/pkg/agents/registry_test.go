package agents

import (
	"testing"
	"time"
)

func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		name    string
		agent   *Agent
		wantErr bool
	}{
		{
			name: "register valid agent",
			agent: &Agent{
				Name:         "test-agent",
				Version:      "v1",
				Description:  "Test agent",
				SystemPrompt: "You are a test agent.",
				LoadedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name:    "register nil agent",
			agent:   nil,
			wantErr: true,
		},
		{
			name: "register agent with empty name",
			agent: &Agent{
				Name:         "",
				Version:      "v1",
				Description:  "Test agent",
				SystemPrompt: "You are a test agent.",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.Register(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistry_Register_Update(t *testing.T) {
	reg := NewRegistry()

	// 注册第一个 agent
	agent1 := &Agent{
		Name:         "test-agent",
		Version:      "v1",
		Description:  "First version",
		SystemPrompt: "You are a test agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent1); err != nil {
		t.Fatalf("Failed to register first agent: %v", err)
	}

	// 注册同名 agent（应该更新）
	agent2 := &Agent{
		Name:         "test-agent",
		Version:      "v2",
		Description:  "Second version",
		SystemPrompt: "You are an updated test agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent2); err != nil {
		t.Fatalf("Failed to register second agent: %v", err)
	}

	// 验证更新
	retrieved, err := reg.Get("test-agent")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if retrieved.Version != "v2" {
		t.Errorf("Expected version v2, got %s", retrieved.Version)
	}

	if retrieved.Description != "Second version" {
		t.Errorf("Expected description 'Second version', got %s", retrieved.Description)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()

	// 注册 agent
	agent := &Agent{
		Name:         "test-agent",
		Version:      "v1",
		Description:  "Test agent",
		SystemPrompt: "You are a test agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// 验证存在
	if !reg.Exists("test-agent") {
		t.Error("Agent should exist after registration")
	}

	// 注销
	if err := reg.Unregister("test-agent"); err != nil {
		t.Errorf("Unregister() unexpected error = %v", err)
	}

	// 验证不存在
	if reg.Exists("test-agent") {
		t.Error("Agent should not exist after unregistration")
	}

	// 再次注销应该报错
	if err := reg.Unregister("test-agent"); err == nil {
		t.Error("Unregister() expected error for non-existent agent, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()

	// 注册 agent
	agent := &Agent{
		Name:         "test-agent",
		Version:      "v1",
		Description:  "Test agent",
		SystemPrompt: "You are a test agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// 获取存在的 agent
	retrieved, err := reg.Get("test-agent")
	if err != nil {
		t.Errorf("Get() unexpected error = %v", err)
	}

	if retrieved.Name != "test-agent" {
		t.Errorf("Get() returned wrong agent, got %s", retrieved.Name)
	}

	// 获取不存在的 agent
	_, err = reg.Get("non-existent")
	if err == nil {
		t.Error("Get() expected error for non-existent agent, got nil")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	// 初始应该为空
	if len(reg.List()) != 0 {
		t.Error("List() should return empty list initially")
	}

	// 注册多个 agents
	agents := []*Agent{
		{Name: "agent-1", Version: "v1", Description: "Agent 1", SystemPrompt: "Prompt 1", LoadedAt: time.Now()},
		{Name: "agent-2", Version: "v1", Description: "Agent 2", SystemPrompt: "Prompt 2", LoadedAt: time.Now()},
		{Name: "agent-3", Version: "v1", Description: "Agent 3", SystemPrompt: "Prompt 3", LoadedAt: time.Now()},
	}

	for _, agent := range agents {
		if err := reg.Register(agent); err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}
	}

	// 列出所有
	list := reg.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d agents, expected 3", len(list))
	}

	// 验证包含所有 agents
	names := make(map[string]bool)
	for _, agent := range list {
		names[agent.Name] = true
	}

	for _, agent := range agents {
		if !names[agent.Name] {
			t.Errorf("List() missing agent %s", agent.Name)
		}
	}
}

func TestRegistry_Exists(t *testing.T) {
	reg := NewRegistry()

	// 检查不存在的 agent
	if reg.Exists("test-agent") {
		t.Error("Exists() should return false for non-existent agent")
	}

	// 注册 agent
	agent := &Agent{
		Name:         "test-agent",
		Version:      "v1",
		Description:  "Test agent",
		SystemPrompt: "You are a test agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// 检查存在的 agent
	if !reg.Exists("test-agent") {
		t.Error("Exists() should return true for existing agent")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()

	// 并发注册
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			agent := &Agent{
				Name:         "agent",
				Version:      "v1",
				Description:  "Test agent",
				SystemPrompt: "You are a test agent.",
				LoadedAt:     time.Now(),
			}
			// 使用同一个 name 来测试并发更新
			reg.Register(agent)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终状态
	if !reg.Exists("agent") {
		t.Error("Agent should exist after concurrent registration")
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		go func() {
			reg.Get("agent")
			reg.List()
			reg.Exists("agent")
		}()
	}
}
