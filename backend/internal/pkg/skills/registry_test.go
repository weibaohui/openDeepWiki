package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// mockSkill 测试用的 Skill 实现
type mockSkill struct {
	name        string
	description string
	params      llm.ParameterSchema
	executeFunc func(ctx context.Context, args json.RawMessage) (interface{}, error)
}

func (m *mockSkill) Name() string {
	return m.name
}

func (m *mockSkill) Description() string {
	return m.description
}

func (m *mockSkill) Parameters() llm.ParameterSchema {
	return m.params
}

func (m *mockSkill) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return nil, nil
}

func (m *mockSkill) ProviderType() string {
	return "mock"
}

func newMockSkill(name string) *mockSkill {
	return &mockSkill{
		name:        name,
		description: "Mock skill for testing",
		params: llm.ParameterSchema{
			Type: "object",
			Properties: map[string]llm.Property{
				"input": {Type: "string", Description: "Input parameter"},
			},
		},
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	skill := newMockSkill("test_skill")

	// 测试注册
	err := r.Register(skill)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// 测试重复注册
	err = r.Register(skill)
	if err == nil {
		t.Error("Register() should return error for duplicate skill")
	}

	// 测试注册 nil
	err = r.Register(nil)
	if err == nil {
		t.Error("Register() should return error for nil skill")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	skill := newMockSkill("test_skill")
	r.Register(skill)

	// 测试注销
	err := r.Unregister("test_skill")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	// 测试注销不存在的
	err = r.Unregister("non_existent")
	if err == nil {
		t.Error("Unregister() should return error for non-existent skill")
	}
}

func TestRegistry_EnableDisable(t *testing.T) {
	r := NewRegistry()

	skill := newMockSkill("test_skill")
	r.Register(skill)

	// 默认应该是启用的
	if !r.IsEnabled("test_skill") {
		t.Error("Newly registered skill should be enabled by default")
	}

	// 测试禁用
	err := r.Disable("test_skill")
	if err != nil {
		t.Errorf("Disable() error = %v", err)
	}
	if r.IsEnabled("test_skill") {
		t.Error("Disabled skill should not be enabled")
	}

	// 测试启用
	err = r.Enable("test_skill")
	if err != nil {
		t.Errorf("Enable() error = %v", err)
	}
	if !r.IsEnabled("test_skill") {
		t.Error("Enabled skill should be enabled")
	}

	// 测试对不存在的 skill 操作
	err = r.Enable("non_existent")
	if err == nil {
		t.Error("Enable() should return error for non-existent skill")
	}

	err = r.Disable("non_existent")
	if err == nil {
		t.Error("Disable() should return error for non-existent skill")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	skill := newMockSkill("test_skill")
	r.Register(skill)

	// 测试获取
	got, err := r.Get("test_skill")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got.Name() != "test_skill" {
		t.Errorf("Get() got = %v, want %v", got.Name(), "test_skill")
	}

	// 测试获取不存在的
	_, err = r.Get("non_existent")
	if err == nil {
		t.Error("Get() should return error for non-existent skill")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// 空列表
	if len(r.List()) != 0 {
		t.Error("Empty registry should return empty list")
	}

	// 添加 skills
	r.Register(newMockSkill("skill1"))
	r.Register(newMockSkill("skill2"))
	r.Register(newMockSkill("skill3"))

	if len(r.List()) != 3 {
		t.Errorf("List() returned %d skills, want 3", len(r.List()))
	}
}

func TestRegistry_ListEnabled(t *testing.T) {
	r := NewRegistry()

	r.Register(newMockSkill("skill1"))
	r.Register(newMockSkill("skill2"))
	r.Register(newMockSkill("skill3"))

	// 禁用其中一个
	r.Disable("skill2")

	enabled := r.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("ListEnabled() returned %d skills, want 2", len(enabled))
	}

	// 检查禁用的不在列表中
	for _, s := range enabled {
		if s.Name() == "skill2" {
			t.Error("Disabled skill should not appear in ListEnabled()")
		}
	}
}

func TestRegistry_ToTools(t *testing.T) {
	r := NewRegistry()

	skill := newMockSkill("test_skill")
	skill.description = "Test description"
	skill.params = llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.Property{
			"param1": {Type: "string", Description: "Parameter 1"},
		},
		Required: []string{"param1"},
	}
	r.Register(skill)

	tools := r.ToTools()
	if len(tools) != 1 {
		t.Errorf("ToTools() returned %d tools, want 1", len(tools))
	}

	tool := tools[0]
	if tool.Type != "function" {
		t.Errorf("Tool.Type = %v, want function", tool.Type)
	}
	if tool.Function.Name != "test_skill" {
		t.Errorf("Tool.Function.Name = %v, want test_skill", tool.Function.Name)
	}
	if tool.Function.Description != "Test description" {
		t.Errorf("Tool.Function.Description = %v, want Test description", tool.Function.Description)
	}

	// 测试禁用后的转换
	r.Disable("test_skill")
	tools = r.ToTools()
	if len(tools) != 0 {
		t.Errorf("ToTools() returned %d tools after disable, want 0", len(tools))
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()

	// 并发注册
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			skill := newMockSkill(fmt.Sprintf("skill_%d", idx))
			r.Register(skill)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if len(r.List()) != 10 {
		t.Errorf("Concurrent registration resulted in %d skills, want 10", len(r.List()))
	}
}
