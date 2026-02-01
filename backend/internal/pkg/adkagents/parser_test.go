package adkagents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	// 创建临时测试文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-agent.yaml")
	
	configContent := `name: TestAgent
description: 测试 Agent

model: ""

instruction: |
  这是一个测试 Agent。

tools:
  - list_dir
  - read_file

maxIterations: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// 测试解析
	agent, err := parser.Parse(configPath)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// 验证字段
	if agent.Name != "TestAgent" {
		t.Errorf("expected name 'TestAgent', got '%s'", agent.Name)
	}
	if agent.Description != "测试 Agent" {
		t.Errorf("expected description '测试 Agent', got '%s'", agent.Description)
	}
	if agent.Model != "" {
		t.Errorf("expected empty model, got '%s'", agent.Model)
	}
	if agent.Instruction == "" {
		t.Error("expected non-empty instruction")
	}
	if len(agent.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(agent.Tools))
	}
	if agent.MaxIterations != 10 {
		t.Errorf("expected maxIterations 10, got %d", agent.MaxIterations)
	}
}

func TestParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		agent   *AgentDefinition
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: &AgentDefinition{
				Name:          "ValidAgent",
				Description:   "A valid agent",
				Instruction:   "Do something useful.",
				MaxIterations: 10,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			agent: &AgentDefinition{
				Name:          "",
				Description:   "An agent with no name",
				Instruction:   "Do something.",
				MaxIterations: 10,
			},
			wantErr: true,
		},
		{
			name: "empty description",
			agent: &AgentDefinition{
				Name:          "NoDescAgent",
				Description:   "",
				Instruction:   "Do something.",
				MaxIterations: 10,
			},
			wantErr: true,
		},
		{
			name: "empty instruction",
			agent: &AgentDefinition{
				Name:          "NoInstrAgent",
				Description:   "An agent with no instruction",
				Instruction:   "",
				MaxIterations: 10,
			},
			wantErr: true,
		},
		{
			name: "invalid maxIterations",
			agent: &AgentDefinition{
				Name:          "InvalidMaxIter",
				Description:   "An agent with invalid maxIterations",
				Instruction:   "Do something.",
				MaxIterations: 0,
			},
			wantErr: true,
		},
		{
			name: "maxIterations too large",
			agent: &AgentDefinition{
				Name:          "LargeMaxIter",
				Description:   "An agent with too large maxIterations",
				Instruction:   "Do something.",
				MaxIterations: 200,
			},
			wantErr: true,
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
		valid bool
	}{
		{"valid-agent", true},
		{"ValidAgent", true},
		{"agent123", true},
		{"agent_123", true},
		{"RepoInitializer", true},
		{"", false},
		{"-invalid", false},
		{"invalid-", false},
		{"invalid--name", false},
		{"invalid name", false},
		{"invalid@name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidAgentName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidAgentName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
