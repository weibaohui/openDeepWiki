package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchFiles(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建测试文件结构
	files := []string{
		"main.go",
		"utils/helper.go",
		"utils/string.go",
		"internal/config/config.go",
		"internal/service/user.go",
		"README.md",
		"go.mod",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	tests := []struct {
		name      string
		pattern   string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "search all  files",
			pattern:   "**/*",
			wantCount: 7,
			wantErr:   false,
		},
		{
			name:      "search all go files",
			pattern:   "**/*.go",
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "search in specific directory",
			pattern:   "utils/*.go",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "search single file",
			pattern:   "main.go",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "search non-existent pattern",
			pattern:   "**/*.py",
			wantCount: 0, // 返回 "No files found" 消息
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
			args, _ := json.Marshal(map[string]string{"pattern": tt.pattern})
			result, err := SearchFiles(args, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchFiles() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("SearchFiles() unexpected error: %v", err)
				return
			}

			if tt.wantCount == 0 {
				if result != "No files found matching the pattern in ." {
					t.Errorf("SearchFiles() expected 'No files found' message, got: %s", result)
				}
			} else {
				// 检查结果中包含的文件数量
				lines := 0
				for _, c := range result {
					if c == '\n' {
						lines++
					}
				}
				// 注意：最后一行可能没有换行符
				if result != "" && result[len(result)-1] != '\n' {
					lines++
				}
				if lines != tt.wantCount {
					t.Errorf("SearchFiles() expected %d files, got %d: %s", tt.wantCount, lines, result)
				}
			}
		})
	}
}

func TestSearchFilesPathSafety(t *testing.T) {
	tempDir := t.TempDir()

	// 测试路径安全
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "normal path",
			path:    "subdir",
			wantErr: false,
		},
		{
			name:    "path traversal attempt",
			path:    "../..",
			wantErr: true,
		},
		{
			name:    "absolute path attempt",
			path:    "/etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]string{
				"pattern": "*.go",
				"path":    tt.path,
			})
			_, err := SearchFiles(args, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SearchFiles() expected error for path %s", tt.path)
				}
			} else {
				// 路径本身安全，但可能不存在，这不是错误
				if err != nil && err.Error() != "search failed: open "+filepath.Join(tempDir, tt.path)+": no such file or directory" {
					t.Errorf("SearchFiles() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGlobSearch(t *testing.T) {
	tempDir := t.TempDir()

	// 创建测试文件
	testFiles := []string{
		"a.go",
		"b.go",
		"c.txt",
		"subdir/d.go",
		"subdir/e.go",
	}

	for _, f := range testFiles {
		path := filepath.Join(tempDir, f)
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0755)
		os.WriteFile(path, []byte("test"), 0644)
	}

	tests := []struct {
		name      string
		pattern   string
		wantCount int
	}{
		{
			name:      "simple glob",
			pattern:   "*.go",
			wantCount: 2,
		},
		{
			name:      "recursive glob",
			pattern:   "**/*.go",
			wantCount: 4,
		},
		{
			name:      "all files",
			pattern:   "*.go",
			wantCount: 2, // a.go, b.go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := globSearch(tempDir, tt.pattern)
			if err != nil {
				t.Errorf("globSearch() unexpected error: %v", err)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("globSearch() expected %d results, got %d: %v", tt.wantCount, len(results), results)
			}
		})
	}
}

func TestIsPathSafe(t *testing.T) {
	tests := []struct {
		name       string
		basePath   string
		targetPath string
		want       bool
	}{
		{
			name:       "safe path within base",
			basePath:   "/home/user/project",
			targetPath: "/home/user/project/main.go",
			want:       true,
		},
		{
			name:       "path traversal attempt",
			basePath:   "/home/user/project",
			targetPath: "/home/user/project/../../etc/passwd",
			want:       false,
		},
		{
			name:       "exact base path",
			basePath:   "/home/user/project",
			targetPath: "/home/user/project",
			want:       true,
		},
		{
			name:       "outside base path",
			basePath:   "/home/user/project",
			targetPath: "/home/other/file.txt",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPathSafe(tt.basePath, tt.targetPath)
			if got != tt.want {
				t.Errorf("isPathSafe(%q, %q) = %v, want %v", tt.basePath, tt.targetPath, got, tt.want)
			}
		})
	}
}
