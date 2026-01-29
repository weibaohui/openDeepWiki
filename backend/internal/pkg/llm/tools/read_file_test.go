package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 创建大文件（超过 1MB）
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("a", 2*1024*1024) // 2MB
	if err := os.WriteFile(largeFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		offset    int
		limit     int
		wantErr   bool
		wantCheck func(string) bool
	}{
		{
			name:    "read entire file",
			path:    "test.txt",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "Line 1") && strings.Contains(s, "Line 5")
			},
		},
		{
			name:    "read with offset",
			path:    "test.txt",
			offset:  3,
			wantErr: false,
			wantCheck: func(s string) bool {
				return !strings.Contains(s, "Line 1") && strings.Contains(s, "Line 3")
			},
		},
		{
			name:    "read with limit",
			path:    "test.txt",
			limit:   2,
			wantErr: false,
			wantCheck: func(s string) bool {
				// 期望得到 2 行内容 + 1 行截断提示 = 3 行
				lines := strings.Split(strings.TrimSpace(s), "\n")
				return len(lines) == 3 && strings.Contains(s, "Line 1")
			},
		},
		{
			name:    "read non-existent file",
			path:    "not_exist.txt",
			wantErr: true,
		},
		{
			name:    "read directory",
			path:    ".",
			wantErr: true,
		},
		{
			name:    "read large file",
			path:    "large.txt",
			wantErr: true, // 超过 1MB 限制
		},
		{
			name:    "path traversal attempt",
			path:    "../test.txt",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := ReadFileArgs{
				Path:   tt.path,
				Offset: tt.offset,
				Limit:  tt.limit,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := ReadFile(argsJSON, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadFile() expected error but got result: %s", result)
				}
				return
			}

			if err != nil {
				t.Errorf("ReadFile() unexpected error: %v", err)
				return
			}

			if tt.wantCheck != nil && !tt.wantCheck(result) {
				t.Errorf("ReadFile() result check failed: %s", result)
			}
		})
	}
}

func TestReadFileLineCount(t *testing.T) {
	tempDir := t.TempDir()

	// 创建包含多行的测试文件
	var lines []string
	for i := 1; i <= 200; i++ {
		lines = append(lines, "Line "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")
	testFile := filepath.Join(tempDir, "lines.txt")
	os.WriteFile(testFile, []byte(content), 0644)

	tests := []struct {
		name     string
		offset   int
		limit    int
		expected int
	}{
		{
			name:     "default limit (100)",
			offset:   1,
			limit:    0,
			expected: 101, // 100 行内容 + 1 行截断提示
		},
		{
			name:     "custom limit 50",
			offset:   1,
			limit:    50,
			expected: 51, // 50 行内容 + 1 行截断提示
		},
		{
			name:     "max limit 500",
			offset:   1,
			limit:    1000, // 超过 500 会被限制
			expected: 200,  // 只有 200 行，不会触发截断
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := ReadFileArgs{
				Path:   "lines.txt",
				Offset: tt.offset,
				Limit:  tt.limit,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := ReadFile(argsJSON, tempDir)
			if err != nil {
				t.Fatalf("ReadFile() unexpected error: %v", err)
			}

			resultLines := strings.Split(strings.TrimSpace(result), "\n")
			if len(resultLines) != tt.expected {
				t.Errorf("ReadFile() expected %d lines, got %d", tt.expected, len(resultLines))
			}
		})
	}
}

func TestReadFileBinary(t *testing.T) {
	tempDir := t.TempDir()

	// 创建包含 null 字节的二进制文件
	binaryContent := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't', 0x00}
	testFile := filepath.Join(tempDir, "binary.bin")
	os.WriteFile(testFile, binaryContent, 0644)

	args := ReadFileArgs{Path: "binary.bin"}
	argsJSON, _ := json.Marshal(args)
	result, err := ReadFile(argsJSON, tempDir)
	if err != nil {
		t.Errorf("ReadFile() should handle binary files: %v", err)
	}
	if result == "" {
		t.Error("ReadFile() should return content for binary files")
	}
}
