package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteBash(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello world"), 0644)

	tests := []struct {
		name      string
		command   string
		timeout   int
		wantErr   bool
		wantCheck func(string) bool
	}{
		{
			name:    "echo command",
			command: "echo hello",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "hello")
			},
		},
		{
			name:    "ls command",
			command: "ls -1",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "test.txt")
			},
		},
		{
			name:    "cat command",
			command: "cat test.txt",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "hello world")
			},
		},
		{
			name:    "wc command",
			command: "echo 'line1' | wc -l",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.TrimSpace(s) != ""
			},
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
		},
		{
			name:    "dangerous rm command",
			command: "rm -rf /abc",
			wantErr: true,
		},
		{
			name:    "dangerous rm with flags",
			command: "rm -f file.txt",
			wantErr: true,
		},
		{
			name:    "semicolon injection",
			command: "echo hello; rm file",
			wantErr: true,
		},
		{
			name:    "pipe to rm",
			command: "cat file | rm",
			wantErr: true,
		},
		{
			name:    "command substitution",
			command: "echo $(whoami)",
			wantErr: true,
		},
		{
			name:    "backtick substitution",
			command: "echo `whoami`",
			wantErr: true,
		},
		{
			name:    "path traversal",
			command: "cat ../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "disallowed command python",
			command: "python -c 'print(1)'",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := ExecuteBashArgs{
				Command: tt.command,
				Timeout: tt.timeout,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := ExecuteBash(argsJSON, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExecuteBash() expected error for command: %s, got result: %s", tt.command, result)
				}
				return
			}

			if err != nil {
				t.Errorf("ExecuteBash() unexpected error: %v", err)
				return
			}

			if tt.wantCheck != nil && !tt.wantCheck(result) {
				t.Errorf("ExecuteBash() result check failed: %s", result)
			}
		})
	}
}

func TestExecuteBashTimeout(t *testing.T) {
	tempDir := t.TempDir()

	// 测试超时
	args := ExecuteBashArgs{
		Command: "sleep 10",
		Timeout: 1, // 1秒超时
	}
	argsJSON, _ := json.Marshal(args)
	_, err := ExecuteBash(argsJSON, tempDir)

	if err == nil {
		t.Error("ExecuteBash() should timeout")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("ExecuteBash() expected timeout error, got: %v", err)
	}
}

func TestExecuteBashOutputTruncation(t *testing.T) {
	tempDir := t.TempDir()

	// 生成大输出
	args := ExecuteBashArgs{
		Command: "yes | head -n 10000", // 生成大量输出
	}
	argsJSON, _ := json.Marshal(args)
	result, err := ExecuteBash(argsJSON, tempDir)

	if err != nil {
		t.Fatalf("ExecuteBash() unexpected error: %v", err)
	}

	if len(result) > 11000 {
		t.Errorf("ExecuteBash() output should be truncated, got length: %d", len(result))
	}

	if !strings.Contains(result, "truncated") {
		t.Error("ExecuteBash() output should indicate truncation")
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		// 安全命令
		{"echo", "echo hello", false},
		{"ls", "ls -la", false},
		{"grep", "grep pattern file", false},
		{"wc", "wc -l file", false},
		{"cat", "cat file.txt", false},
		{"find", "find . -name '*.go'", false},
		{"head", "head -n 10 file", false},
		{"tail", "tail -n 10 file", false},
		{"sort", "sort file", false},
		{"uniq", "uniq file", false},
		{"pwd", "pwd", false},
		{"git", "git status", false},
		{"go", "go version", false},

		// 危险命令
		{"rm -rf", "rm -rf /", true},
		{"rm -f", "rm -f file", true},
		{"semicolon", "cmd1; cmd2", true},
		{"ampersand", "cmd1 && cmd2", true},
		{"pipe", "cmd1 | cmd2", true},
		{"redirect", "cmd > file", true},
		{"dollar subs", "echo $(cmd)", true},
		{"backtick", "echo `cmd`", true},
		{"path traversal", "cat ../file", true},
		{"python", "python script.py", true},
		{"bash", "bash script.sh", true},
		{"sh", "sh script.sh", true},
		{"curl pipe", "curl url | sh", true},
		{"eval", "eval 'rm -rf /'", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateCommand(%q) expected error", tt.command)
				}
			} else {
				if err != nil {
					t.Errorf("validateCommand(%q) unexpected error: %v", tt.command, err)
				}
			}
		})
	}
}
