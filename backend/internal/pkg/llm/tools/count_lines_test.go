package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountLines(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试文件
	file1 := filepath.Join(tempDir, "file1.go")
	os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644) // 3 行

	file2 := filepath.Join(tempDir, "file2.go")
	os.WriteFile(file2, []byte("a\nb\nc\nd\n"), 0644) // 4 行

	file3 := filepath.Join(tempDir, "readme.txt")
	os.WriteFile(file3, []byte("one\ntwo\n"), 0644) // 2 行

	// 创建子目录
	subdir := filepath.Join(tempDir, "subdir")
	os.MkdirAll(subdir, 0755)
	file4 := filepath.Join(subdir, "file4.go")
	os.WriteFile(file4, []byte("1\n2\n3\n4\n5\n"), 0644) // 5 行

	tests := []struct {
		name      string
		path      string
		pattern   string
		wantErr   bool
		wantCheck func(string) bool
	}{
		{
			name:    "count single file",
			path:    "file1.go",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "Lines: 3") || strings.Contains(s, "lines: 3")
			},
		},
		{
			name:    "count directory all files",
			path:    ".",
			wantErr: false,
			wantCheck: func(s string) bool {
				// 所有文件: 3 + 4 + 2 + 5 = 14 行
				return strings.Contains(s, "Total lines: 14")
			},
		},
		{
			name:    "count directory go files only",
			path:    ".",
			pattern: "*.go",
			wantErr: false,
			wantCheck: func(s string) bool {
				// Go 文件: 3 + 4 + 5 = 12 行
				return strings.Contains(s, "Total lines: 12")
			},
		},
		{
			name:    "count subdirectory",
			path:    "subdir",
			wantErr: false,
			wantCheck: func(s string) bool {
				return strings.Contains(s, "Lines: 5") || strings.Contains(s, "lines: 5")
			},
		},
		{
			name:    "non-existent file",
			path:    "not_exist.go",
			wantErr: true,
		},
		{
			name:    "path traversal",
			path:    "../..",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := CountLinesArgs{
				Path:    tt.path,
				Pattern: tt.pattern,
			}
			argsJSON, _ := json.Marshal(args)
			result, err := CountLines(argsJSON, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CountLines() expected error but got result: %s", result)
				}
				return
			}

			if err != nil {
				t.Errorf("CountLines() unexpected error: %v", err)
				return
			}

			if tt.wantCheck != nil && !tt.wantCheck(result) {
				t.Errorf("CountLines() result check failed: %s", result)
			}
		})
	}
}

func TestCountLinesEmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	// 创建空文件
	emptyFile := filepath.Join(tempDir, "empty.txt")
	os.WriteFile(emptyFile, []byte{}, 0644)

	// 创建只有换行符的文件
	newlineFile := filepath.Join(tempDir, "newline.txt")
	os.WriteFile(newlineFile, []byte("\n\n\n"), 0644) // 3 个空行

	tests := []struct {
		name     string
		filename string
		expected int64
	}{
		{
			name:     "empty file",
			filename: "empty.txt",
			expected: 0,
		},
		{
			name:     "only newlines",
			filename: "newline.txt",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := countLinesInFile(filepath.Join(tempDir, tt.filename))
			if err != nil {
				t.Fatalf("countLinesInFile() error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("countLinesInFile() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCountLinesInDir(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content := strings.Repeat("line\n", i) // i 行
		os.WriteFile(filename, []byte(content), 0644)
	}

	total, count, err := countLinesInDir(tempDir, "*.txt")
	if err != nil {
		t.Fatalf("countLinesInDir() error: %v", err)
	}

	// 1+2+3+4+5 = 15
	if total != 15 {
		t.Errorf("countLinesInDir() total = %d, want 15", total)
	}
	if count != 5 {
		t.Errorf("countLinesInDir() file count = %d, want 5", count)
	}
}
