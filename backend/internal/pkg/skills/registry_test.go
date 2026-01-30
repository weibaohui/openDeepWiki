package skills

import (
	"fmt"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
	}

	// 测试注册
	err := r.Register(skill)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// 测试重复注册（应该更新）
	skill2 := &Skill{
		Name:        "test-skill",
		Description: "Updated description",
	}
	err = r.Register(skill2)
	if err != nil {
		t.Errorf("Register() update error = %v", err)
	}

	// 验证更新
	got, _ := r.Get("test-skill")
	if got.Description != "Updated description" {
		t.Errorf("Description not updated, got = %v", got.Description)
	}

	// 测试注册 nil
	err = r.Register(nil)
	if err == nil {
		t.Error("Register() should return error for nil skill")
	}

	// 测试空 name
	err = r.Register(&Skill{Name: ""})
	if err == nil {
		t.Error("Register() should return error for empty name")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
	}
	r.Register(skill)

	// 测试注销
	err := r.Unregister("test-skill")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	// 测试注销不存在的
	err = r.Unregister("non-existent")
	if err == nil {
		t.Error("Unregister() should return error for non-existent skill")
	}
}

func TestRegistry_EnableDisable(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
	}
	r.Register(skill)

	// 默认应该是启用的
	if !r.IsEnabled("test-skill") {
		t.Error("Newly registered skill should be enabled by default")
	}

	// 测试禁用
	err := r.Disable("test-skill")
	if err != nil {
		t.Errorf("Disable() error = %v", err)
	}
	if r.IsEnabled("test-skill") {
		t.Error("Disabled skill should not be enabled")
	}

	// 测试启用
	err = r.Enable("test-skill")
	if err != nil {
		t.Errorf("Enable() error = %v", err)
	}
	if !r.IsEnabled("test-skill") {
		t.Error("Enabled skill should be enabled")
	}

	// 测试对不存在的 skill 操作
	err = r.Enable("non-existent")
	if err == nil {
		t.Error("Enable() should return error for non-existent skill")
	}

	err = r.Disable("non-existent")
	if err == nil {
		t.Error("Disable() should return error for non-existent skill")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
	}
	r.Register(skill)

	// 测试获取
	got, err := r.Get("test-skill")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got.Name != "test-skill" {
		t.Errorf("Get() got = %v, want test-skill", got.Name)
	}

	// 测试获取不存在的
	_, err = r.Get("non-existent")
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
	r.Register(&Skill{Name: "skill1", Description: "desc1"})
	r.Register(&Skill{Name: "skill2", Description: "desc2"})
	r.Register(&Skill{Name: "skill3", Description: "desc3"})

	if len(r.List()) != 3 {
		t.Errorf("List() returned %d skills, want 3", len(r.List()))
	}
}

func TestRegistry_ListEnabled(t *testing.T) {
	r := NewRegistry()

	r.Register(&Skill{Name: "skill1", Description: "desc1"})
	r.Register(&Skill{Name: "skill2", Description: "desc2"})
	r.Register(&Skill{Name: "skill3", Description: "desc3"})

	// 禁用其中一个
	r.Disable("skill2")

	enabled := r.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("ListEnabled() returned %d skills, want 2", len(enabled))
	}

	// 检查禁用的不在列表中
	for _, s := range enabled {
		if s.Name == "skill2" {
			t.Error("Disabled skill should not appear in ListEnabled()")
		}
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()

	// 并发注册
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			skill := &Skill{
				Name:        fmt.Sprintf("skill_%d", idx),
				Description: fmt.Sprintf("Description %d", idx),
			}
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
