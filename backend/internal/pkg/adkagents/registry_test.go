package adkagents

import (
	"testing"
	"time"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	agent := &AgentDefinition{
		Name:        "TestAgent",
		Description: "Test Description",
		Instruction: "Test Instruction",
		LoadedAt:    Now(),
	}

	// 测试注册
	err := registry.Register(agent)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 验证存在
	if !registry.Exists("TestAgent") {
		t.Error("agent should exist after registration")
	}

	// 验证获取
	got, err := registry.Get("TestAgent")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if got.Name != "TestAgent" {
		t.Errorf("expected name 'TestAgent', got '%s'", got.Name)
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	agent1 := &AgentDefinition{
		Name:        "TestAgent",
		Description: "First Description",
		Instruction: "First Instruction",
		LoadedAt:    Now(),
	}

	agent2 := &AgentDefinition{
		Name:        "TestAgent",
		Description: "Second Description",
		Instruction: "Second Instruction",
		LoadedAt:    Now(),
	}

	// 注册第一个
	if err := registry.Register(agent1); err != nil {
		t.Fatalf("failed to register first agent: %v", err)
	}

	// 注册同名 Agent（应该覆盖）
	if err := registry.Register(agent2); err != nil {
		t.Fatalf("failed to register second agent: %v", err)
	}

	// 验证已更新
	got, _ := registry.Get("TestAgent")
	if got.Description != "Second Description" {
		t.Error("agent should be updated")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	agent := &AgentDefinition{
		Name:        "TestAgent",
		Description: "Test Description",
		Instruction: "Test Instruction",
		LoadedAt:    Now(),
	}

	registry.Register(agent)

	// 注销
	err := registry.Unregister("TestAgent")
	if err != nil {
		t.Fatalf("failed to unregister agent: %v", err)
	}

	// 验证不存在
	if registry.Exists("TestAgent") {
		t.Error("agent should not exist after unregistration")
	}

	// 再次注销应该报错
	err = registry.Unregister("TestAgent")
	if err == nil {
		t.Error("expected error when unregistering non-existent agent")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// 注册多个 Agent
	agents := []*AgentDefinition{
		{Name: "Agent1", Description: "Desc1", Instruction: "Instr1", LoadedAt: Now()},
		{Name: "Agent2", Description: "Desc2", Instruction: "Instr2", LoadedAt: Now()},
		{Name: "Agent3", Description: "Desc3", Instruction: "Instr3", LoadedAt: Now()},
	}

	for _, agent := range agents {
		if err := registry.Register(agent); err != nil {
			t.Fatalf("failed to register agent: %v", err)
		}
	}

	// 验证列表
	list := registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 agents, got %d", len(list))
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}

	registry.Register(&AgentDefinition{
		Name:        "Agent1",
		Description: "Desc1",
		Instruction: "Instr1",
		LoadedAt:    Now(),
	})

	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&AgentDefinition{
		Name:        "Agent1",
		Description: "Desc1",
		Instruction: "Instr1",
		LoadedAt:    Now(),
	})

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", registry.Count())
	}
}

// 重置 Now 函数为默认实现
func init() {
	Now = func() time.Time {
		return time.Now()
	}
}
