package agents

import (
	"testing"
)

func TestAgent_IsSkillAllowed(t *testing.T) {
	tests := []struct {
		name       string
		skillPolicy SkillPolicy
		skillName  string
		want       bool
	}{
		{
			name:       "empty allow list - skill not in deny",
			skillPolicy: SkillPolicy{},
			skillName:  "any-skill",
			want:       true,
		},
		{
			name: "skill in allow list",
			skillPolicy: SkillPolicy{
				Allow: []string{"skill-1", "skill-2"},
			},
			skillName: "skill-1",
			want:      true,
		},
		{
			name: "skill not in allow list",
			skillPolicy: SkillPolicy{
				Allow: []string{"skill-1", "skill-2"},
			},
			skillName: "skill-3",
			want:      false,
		},
		{
			name: "skill in deny list - takes priority",
			skillPolicy: SkillPolicy{
				Allow: []string{"skill-1", "skill-2"},
				Deny:  []string{"skill-1"},
			},
			skillName: "skill-1",
			want:      false,
		},
		{
			name: "skill in deny list - not in allow",
			skillPolicy: SkillPolicy{
				Allow: []string{"skill-1"},
				Deny:  []string{"skill-2"},
			},
			skillName: "skill-2",
			want:      false,
		},
		{
			name: "empty allow with deny - skill not denied",
			skillPolicy: SkillPolicy{
				Deny: []string{"dangerous-skill"},
			},
			skillName: "safe-skill",
			want:      true,
		},
		{
			name: "empty allow with deny - skill denied",
			skillPolicy: SkillPolicy{
				Deny: []string{"dangerous-skill"},
			},
			skillName: "dangerous-skill",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				SkillPolicy: tt.skillPolicy,
			}
			got := agent.IsSkillAllowed(tt.skillName)
			if got != tt.want {
				t.Errorf("IsSkillAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_IsMcpAllowed(t *testing.T) {
	tests := []struct {
		name      string
		mcpPolicy McpPolicy
		mcpName   string
		want      bool
	}{
		{
			name:      "empty allowed list - allow all",
			mcpPolicy: McpPolicy{},
			mcpName:   "any-mcp",
			want:      true,
		},
		{
			name: "mcp in allowed list",
			mcpPolicy: McpPolicy{
				Allowed: []string{"mcp-1", "mcp-2"},
			},
			mcpName: "mcp-1",
			want:    true,
		},
		{
			name: "mcp not in allowed list",
			mcpPolicy: McpPolicy{
				Allowed: []string{"mcp-1", "mcp-2"},
			},
			mcpName: "mcp-3",
			want:    false,
		},
		{
			name: "mcp allowed list empty - allow all",
			mcpPolicy: McpPolicy{
				MaxCalls: 5,
			},
			mcpName: "any-mcp",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				McpPolicy: tt.mcpPolicy,
			}
			got := agent.IsMcpAllowed(tt.mcpName)
			if got != tt.want {
				t.Errorf("IsMcpAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_Struct(t *testing.T) {
	agent := &Agent{
		Name:         "test-agent",
		Version:      "v1",
		Description:  "Test agent description",
		SystemPrompt: "You are a test agent.",
		McpPolicy: McpPolicy{
			Allowed:  []string{"mcp-1", "mcp-2"},
			MaxCalls: 5,
		},
		SkillPolicy: SkillPolicy{
			Allow: []string{"skill-1", "skill-2"},
			Deny:  []string{"dangerous-skill"},
		},
		RuntimePolicy: RuntimePolicy{
			RiskLevel:           "read",
			MaxSteps:            10,
			RequireConfirmation: true,
		},
	}

	// 验证基本字段
	if agent.Name != "test-agent" {
		t.Errorf("Name = %v, want test-agent", agent.Name)
	}

	if agent.Version != "v1" {
		t.Errorf("Version = %v, want v1", agent.Version)
	}

	if agent.Description != "Test agent description" {
		t.Errorf("Description = %v, want 'Test agent description'", agent.Description)
	}

	if agent.SystemPrompt != "You are a test agent." {
		t.Errorf("SystemPrompt = %v, want 'You are a test agent.'", agent.SystemPrompt)
	}

	// 验证 MCP Policy
	if len(agent.McpPolicy.Allowed) != 2 {
		t.Errorf("McpPolicy.Allowed length = %v, want 2", len(agent.McpPolicy.Allowed))
	}

	if agent.McpPolicy.MaxCalls != 5 {
		t.Errorf("McpPolicy.MaxCalls = %v, want 5", agent.McpPolicy.MaxCalls)
	}

	// 验证 Skill Policy
	if len(agent.SkillPolicy.Allow) != 2 {
		t.Errorf("SkillPolicy.Allow length = %v, want 2", len(agent.SkillPolicy.Allow))
	}

	if len(agent.SkillPolicy.Deny) != 1 {
		t.Errorf("SkillPolicy.Deny length = %v, want 1", len(agent.SkillPolicy.Deny))
	}

	// 验证 Runtime Policy
	if agent.RuntimePolicy.RiskLevel != "read" {
		t.Errorf("RuntimePolicy.RiskLevel = %v, want read", agent.RuntimePolicy.RiskLevel)
	}

	if agent.RuntimePolicy.MaxSteps != 10 {
		t.Errorf("RuntimePolicy.MaxSteps = %v, want 10", agent.RuntimePolicy.MaxSteps)
	}

	if !agent.RuntimePolicy.RequireConfirmation {
		t.Error("RuntimePolicy.RequireConfirmation should be true")
	}
}

func TestMcpPolicy_Struct(t *testing.T) {
	policy := McpPolicy{
		Allowed:  []string{"cluster_state", "pod_logs", "metrics"},
		MaxCalls: 10,
	}

	if len(policy.Allowed) != 3 {
		t.Errorf("Allowed length = %v, want 3", len(policy.Allowed))
	}

	if policy.MaxCalls != 10 {
		t.Errorf("MaxCalls = %v, want 10", policy.MaxCalls)
	}
}

func TestSkillPolicy_Struct(t *testing.T) {
	policy := SkillPolicy{
		Allow: []string{"search_logs", "analyze_logs"},
		Deny:  []string{"restart_pod", "delete_resource"},
	}

	if len(policy.Allow) != 2 {
		t.Errorf("Allow length = %v, want 2", len(policy.Allow))
	}

	if len(policy.Deny) != 2 {
		t.Errorf("Deny length = %v, want 2", len(policy.Deny))
	}
}

func TestRuntimePolicy_Struct(t *testing.T) {
	policy := RuntimePolicy{
		RiskLevel:           "write",
		MaxSteps:            20,
		RequireConfirmation: true,
	}

	if policy.RiskLevel != "write" {
		t.Errorf("RiskLevel = %v, want write", policy.RiskLevel)
	}

	if policy.MaxSteps != 20 {
		t.Errorf("MaxSteps = %v, want 20", policy.MaxSteps)
	}

	if !policy.RequireConfirmation {
		t.Error("RequireConfirmation should be true")
	}
}
