package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchText(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试文件
	files := map[string]string{
		"main.go":     "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
		"utils.go":    "package main\n\nfunc helper() {\n\t// helper function\n}",
		"test.txt":    "This is a test file\nwith multiple lines\n",
		"config.yaml": "database:\n  host: localhost\n",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	tests := []struct {
		name      string
		pattern   string
		glob      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "search in go files for func keyword",
			pattern:   "^func ",
			glob:      "*.go",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "search with regex",
			pattern:   "func.*\\(\\)",
			glob:      "*.go",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "search specific text",
			pattern:   "hello",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "search no match",
			pattern:   "nonexistent",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "search for package keyword",
			pattern:   "^package ",
			glob:      "*.go",
			wantCount: 2, // main.go 和 utils.go
			wantErr:   false,
		},
		{
			name:      "empty pattern",
			pattern:   "",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := SearchTextArgs{
				Pattern: tt.pattern,
				Glob:    tt.glob,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := SearchText(argsJSON, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchText() expected error but got result: %s", result)
				}
				return
			}

			if err != nil {
				t.Errorf("SearchText() unexpected error: %v", err)
				return
			}

			if tt.wantCount == 0 {
				if result != "No matches found." {
					t.Errorf("SearchText() expected 'No matches found' message, got: %s", result)
				}
			} else {
				// 检查结果中的匹配数量（每行一个匹配）
				lines := strings.Count(result, "\n")
				// 考虑结果可能被截断的情况
				if lines < tt.wantCount && !strings.Contains(result, "truncated") {
					t.Errorf("SearchText() expected at least %d matches, got %d: %s", tt.wantCount, lines, result)
				}
			}
		})
	}
}

func TestSearchTextPathSafety(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "path traversal",
			path:    "../..",
			wantErr: true,
		},
		{
			name:    "path traversal with file",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := SearchTextArgs{
				Pattern: "test",
				Path:    tt.path,
			}
			argsJSON, _ := json.Marshal(args)
			_, err := SearchText(argsJSON, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchText() expected error for path %s", tt.path)
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxLen  int
		want    string
	}{
		{
			name:    "no truncation needed",
			input:   "short text",
			maxLen:  20,
			want:    "short text",
		},
		{
			name:    "truncation needed",
			input:   "this is a very long text that needs to be truncated",
			maxLen:  10,
			want:    "this is a ...",
		},
		{
			name:    "exact length",
			input:   "exactlyten",
			maxLen:  10,
			want:    "exactlyten",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
