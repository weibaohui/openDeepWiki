package agents

import (
	"testing"
	"time"
)

func TestRouter_Route(t *testing.T) {
	// 创建 registry 并注册测试 agents
	reg := NewRegistry()

	agents := []*Agent{
		{
			Name:         "diagnose-agent",
			Version:      "v1",
			Description:  "Diagnose agent",
			SystemPrompt: "You are a diagnose agent.",
			LoadedAt:     time.Now(),
		},
		{
			Name:         "ops-agent",
			Version:      "v1",
			Description:  "Ops agent",
			SystemPrompt: "You are an ops agent.",
			LoadedAt:     time.Now(),
		},
		{
			Name:         "default-agent",
			Version:      "v1",
			Description:  "Default agent",
			SystemPrompt: "You are a default agent.",
			LoadedAt:     time.Now(),
		},
	}

	for _, agent := range agents {
		if err := reg.Register(agent); err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}
	}

	router := NewRouter(reg)
	router.RegisterRoute("diagnose", "diagnose-agent")
	router.RegisterRoute("ops", "ops-agent")
	router.SetDefault("default-agent")

	tests := []struct {
		name    string
		ctx     RouterContext
		want    string
		wantErr bool
	}{
		{
			name: "route by explicit name",
			ctx: RouterContext{
				AgentName: "ops-agent",
			},
			want:    "ops-agent",
			wantErr: false,
		},
		{
			name: "route by entry point - diagnose",
			ctx: RouterContext{
				EntryPoint: "diagnose",
			},
			want:    "diagnose-agent",
			wantErr: false,
		},
		{
			name: "route by entry point - ops",
			ctx: RouterContext{
				EntryPoint: "ops",
			},
			want:    "ops-agent",
			wantErr: false,
		},
		{
			name: "route to default",
			ctx: RouterContext{
				EntryPoint: "unknown",
			},
			want:    "default-agent",
			wantErr: false,
		},
		{
			name:    "empty context",
			ctx:     RouterContext{},
			want:    "default-agent",
			wantErr: false,
		},
		{
			name: "explicit name takes priority over entry point",
			ctx: RouterContext{
				AgentName:  "ops-agent",
				EntryPoint: "diagnose",
			},
			want:    "ops-agent",
			wantErr: false,
		},
		{
			name: "non-existent explicit name",
			ctx: RouterContext{
				AgentName: "non-existent",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := router.Route(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name != tt.want {
				t.Errorf("Route() got = %v, want %v", got.Name, tt.want)
			}
		})
	}
}

func TestRouter_Route_NoDefault(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter(reg)

	// 没有注册任何 agent，也没有设置默认
	ctx := RouterContext{
		EntryPoint: "unknown",
	}

	_, err := router.Route(ctx)
	if err == nil {
		t.Error("Route() expected error when no agent found, got nil")
	}
}

func TestRouter_SetDefault(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter(reg)

	// 注册 agent
	agent := &Agent{
		Name:         "default-agent",
		Version:      "v1",
		Description:  "Default agent",
		SystemPrompt: "You are a default agent.",
		LoadedAt:     time.Now(),
	}

	if err := reg.Register(agent); err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// 设置默认
	if err := router.SetDefault("default-agent"); err != nil {
		t.Errorf("SetDefault() unexpected error = %v", err)
	}

	// 验证默认设置成功
	if router.GetDefault() != "default-agent" {
		t.Errorf("GetDefault() = %v, want default-agent", router.GetDefault())
	}

	// 设置不存在的 agent 应该报错
	if err := router.SetDefault("non-existent"); err == nil {
		t.Error("SetDefault() expected error for non-existent agent, got nil")
	}
}

func TestRouter_RegisterRoute(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter(reg)

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

	// 注册路由规则
	router.RegisterRoute("test-entry", "test-agent")

	// 验证路由规则
	got, exists := router.GetRoute("test-entry")
	if !exists {
		t.Error("GetRoute() should return exists = true")
	}
	if got != "test-agent" {
		t.Errorf("GetRoute() = %v, want test-agent", got)
	}

	// 获取不存在的路由
	_, exists = router.GetRoute("non-existent")
	if exists {
		t.Error("GetRoute() should return exists = false for non-existent route")
	}
}

func TestRouter_Route_RouteButAgentNotFound(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter(reg)

	// 注册路由规则，但不注册对应的 agent
	router.RegisterRoute("test", "non-existent-agent")

	ctx := RouterContext{
		EntryPoint: "test",
	}

	_, err := router.Route(ctx)
	if err == nil {
		t.Error("Route() expected error when route found but agent not found, got nil")
	}
}

func TestRouter_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter(reg)

	// 注册 agents
	for i := 0; i < 5; i++ {
		agent := &Agent{
			Name:         "agent-" + string(rune('0'+i)),
			Version:      "v1",
			Description:  "Test agent",
			SystemPrompt: "You are a test agent.",
			LoadedAt:     time.Now(),
		}
		reg.Register(agent)
	}

	// 并发注册路由
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			router.RegisterRoute("entry-"+string(rune('0'+idx)), "agent-0")
			done <- true
		}(i % 5)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 并发路由
	for i := 0; i < 10; i++ {
		go func(idx int) {
			router.Route(RouterContext{
				EntryPoint: "entry-" + string(rune('0'+idx%5)),
			})
		}(i)
	}
}
