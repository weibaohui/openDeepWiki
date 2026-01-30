package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	// 保存原始 Now 函数
	originalNow := Now
	defer func() { Now = originalNow }()

	// 设置固定的测试时间
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	Now = func() time.Time { return fixedTime }

	parser := NewParser()

	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid agent config",
			content: `name: test-agent
version: v1
description: Test agent description
systemPrompt: |
  You are a test agent.
mcp:
  allowed:
    - test-mcp
  maxCalls: 5
skills:
  allow:
    - test-skill
  deny:
    - dangerous-skill
policies:
  riskLevel: read
  maxSteps: 10
  requireConfirmation: false
`,
			wantErr: false,
		},
		{
			name: "missing name",
			content: `version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name: "missing version",
			content: `name: test-agent
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "version is required",
		},
		{
			name: "missing description",
			content: `name: test-agent
version: v1
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "description is required",
		},
		{
			name: "missing systemPrompt",
			content: `name: test-agent
version: v1
description: Test agent
`,
			wantErr:     true,
			errContains: "systemPrompt is required",
		},
		{
			name: "invalid name - uppercase",
			content: `name: Test-Agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name must contain only lowercase letters",
		},
		{
			name: "invalid name - starts with hyphen",
			content: `name: -test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name must contain only lowercase letters",
		},
		{
			name: "invalid name - ends with hyphen",
			content: `name: test-agent-
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name must contain only lowercase letters",
		},
		{
			name: "invalid name - double hyphen",
			content: `name: test--agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name must contain only lowercase letters",
		},
		{
			name: "invalid version format",
			content: `name: test-agent
version: 1.0
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "version must be valid semantic version",
		},
		{
			name: "valid version - v1",
			content: `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr: false,
		},
		{
			name: "valid version - v1.0",
			content: `name: test-agent
version: v1.0
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr: false,
		},
		{
			name: "valid version - v1.0.0",
			content: `name: test-agent
version: v1.0.0
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr: false,
		},
		{
			name: "invalid riskLevel",
			content: `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
policies:
  riskLevel: invalid
`,
			wantErr:     true,
			errContains: "riskLevel must be one of",
		},
		{
			name: "valid riskLevel - read",
			content: `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
policies:
  riskLevel: read
`,
			wantErr: false,
		},
		{
			name: "valid riskLevel - write",
			content: `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
policies:
  riskLevel: write
`,
			wantErr: false,
		},
		{
			name: "valid riskLevel - admin",
			content: `name: test-agent
version: v1
description: Test agent
systemPrompt: You are a test agent.
policies:
  riskLevel: admin
`,
			wantErr: false,
		},
		{
			name: "description too long",
			content: `name: test-agent
version: v1
description: ` + strings.Repeat("a", 1100) + `
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "description exceeds",
		},
		{
			name: "name too long",
			content: `name: ` + strings.Repeat("a", 70) + `
version: v1
description: Test agent
systemPrompt: You are a test agent.
`,
			wantErr:     true,
			errContains: "name exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "agent.yaml")

			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// 解析
			agent, err := parser.Parse(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error but got nil")
				} else if tt.errContains != "" {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("Parse() error = %v, want error containing %v", err, tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Parse() unexpected error = %v", err)
				} else {
					// 验证基本字段
					if agent.Path != configPath {
						t.Errorf("Parse() Path = %v, want %v", agent.Path, configPath)
					}
					if !agent.LoadedAt.Equal(fixedTime) {
						t.Errorf("Parse() LoadedAt = %v, want %v", agent.LoadedAt, fixedTime)
					}
				}
			}
		})
	}
}

func TestParser_Parse_NonExistentFile(t *testing.T) {
	parser := NewParser()

	_, err := parser.Parse("/non/existent/path/agent.yaml")
	if err == nil {
		t.Error("Parse() expected error for non-existent file, got nil")
	}

	if !contains(err.Error(), "config file not found") {
		t.Errorf("Parse() error should contain 'config file not found', got: %v", err)
	}
}

func TestParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		agent   *Agent
		wantErr bool
	}{
		{
			name: "nil agent",
			agent: &Agent{
				Name:         "test",
				Version:      "v1",
				Description:  "Test",
				SystemPrompt: "You are a test agent.",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			agent: &Agent{
				Name:         "",
				Version:      "v1",
				Description:  "Test",
				SystemPrompt: "You are a test agent.",
			},
			wantErr: true,
		},
		{
			name: "valid agent with all policies",
			agent: &Agent{
				Name:         "test-agent",
				Version:      "v1",
				Description:  "Test agent",
				SystemPrompt: "You are a test agent.",
				McpPolicy: McpPolicy{
					Allowed:  []string{"mcp1", "mcp2"},
					MaxCalls: 5,
				},
				SkillPolicy: SkillPolicy{
					Allow: []string{"skill1"},
					Deny:  []string{"skill2"},
				},
				RuntimePolicy: RuntimePolicy{
					RiskLevel:           "read",
					MaxSteps:            10,
					RequireConfirmation: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidAgentName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "test", true},
		{"valid with hyphen", "test-agent", true},
		{"valid with number", "agent123", true},
		{"valid complex", "my-test-agent-123", true},
		{"empty", "", false},
		{"uppercase", "Test-Agent", false},
		{"starts with hyphen", "-test", false},
		{"ends with hyphen", "test-", false},
		{"double hyphen", "test--agent", false},
		{"space", "test agent", false},
		{"underscore", "test_agent", false},
		{"special chars", "test@agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidAgentName(tt.input)
			if got != tt.want {
				t.Errorf("isValidAgentName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"valid v1", "v1", true},
		{"valid v1.0", "v1.0", true},
		{"valid v1.0.0", "v1.0.0", true},
		{"valid v10.20.30", "v10.20.30", true},
		{"empty", "", false},
		{"no v prefix", "1.0.0", false},
		{"double v", "vv1.0.0", false},
		{"invalid format", "v1.0.0.0", false},
		{"alpha in version", "v1.a.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidVersion(tt.version)
			if got != tt.want {
				t.Errorf("isValidVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
