package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsPathSafe(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "base")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}

	tests := []struct {
		name       string
		basePath   string
		targetPath string
		want       bool
	}{
		{
			name:       "target is base directory",
			basePath:   basePath,
			targetPath: basePath,
			want:       true,
		},
		{
			name:       "target is subdirectory",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, "subdir"),
			want:       true,
		},
		{
			name:       "target is nested file",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, "subdir", "file.txt"),
			want:       true,
		},
		{
			name:       "target escapes with ../",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, "subdir", "..", "..", "etc"),
			want:       false,
		},
		{
			name:       "target is sibling directory",
			basePath:   basePath,
			targetPath: filepath.Join(tempDir, "other"),
			want:       false,
		},
		{
			name:       "target has similar prefix",
			basePath:   basePath,
			targetPath: basePath + "_backup",
			want:       false,
		},
		{
			name:       "target is absolute path outside",
			basePath:   basePath,
			targetPath: "/etc/passwd",
			want:       false,
		},
		{
			name:       "target uses relative path within",
			basePath:   basePath,
			targetPath: ".",
			want:       true,
		},
		{
			name:       "target uses relative subdirectory",
			basePath:   basePath,
			targetPath: "./subdir",
			want:       true,
		},
		{
			name:       "target is parent directory",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, ".."),
			want:       false,
		},
		{
			name:       "target traverses up and back",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, "..", filepath.Base(basePath), "file.txt"),
			want:       true, // 这实际上是安全的，因为最终仍在 basePath 内
		},
		{
			name:       "target traverses up and stays out",
			basePath:   basePath,
			targetPath: filepath.Join(basePath, "..", "other"),
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

func TestIsPathSafe_WithSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink tests on Windows")
	}

	// 创建临时目录用于测试
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "base")
	outsidePath := filepath.Join(tempDir, "outside")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}
	if err := os.MkdirAll(outsidePath, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	// 创建指向外部目录的符号链接
	symlinkPath := filepath.Join(basePath, "link")
	if err := os.Symlink(outsidePath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// 通过符号链接访问应该被拒绝
	targetPath := filepath.Join(symlinkPath, "file.txt")
	if isPathSafe(basePath, targetPath) {
		t.Errorf("isPathSafe should reject paths through symlinks pointing outside base: %q", targetPath)
	}

	// 创建指向内部目录的符号链接
	insidePath := filepath.Join(basePath, "subdir")
	if err := os.MkdirAll(insidePath, 0755); err != nil {
		t.Fatalf("Failed to create inside directory: %v", err)
	}
	insideLink := filepath.Join(basePath, "inside_link")
	if err := os.Symlink(insidePath, insideLink); err != nil {
		t.Fatalf("Failed to create inside symlink: %v", err)
	}

	// 通过内部符号链接访问应该被允许
	targetPath = filepath.Join(insideLink, "file.txt")
	if !isPathSafe(basePath, targetPath) {
		t.Errorf("isPathSafe should allow paths through symlinks pointing inside base: %q", targetPath)
	}
}

func TestValidateAndResolvePath(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "base")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}

	tests := []struct {
		name      string
		inputPath string
		wantErr   bool
	}{
		{
			name:      "valid relative path",
			inputPath: "subdir/file.txt",
			wantErr:   false,
		},
		{
			name:      "valid simple path",
			inputPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "path with traversal",
			inputPath: "../etc/passwd",
			wantErr:   true,
		},
		{
			name:      "nested path with traversal",
			inputPath: "subdir/../../../etc/passwd",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndResolvePath(basePath, tt.inputPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAndResolvePath(%q) expected error, got result: %q", tt.inputPath, result)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAndResolvePath(%q) unexpected error: %v", tt.inputPath, err)
				}
				if result == "" {
					t.Errorf("ValidateAndResolvePath(%q) returned empty result", tt.inputPath)
				}
			}
		})
	}
}

func TestValidateWorkingDir(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "base")
	subPath := filepath.Join(basePath, "subdir")
	outsidePath := filepath.Join(tempDir, "outside")

	if err := os.MkdirAll(subPath, 0755); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	if err := os.MkdirAll(outsidePath, 0755); err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}

	tests := []struct {
		name      string
		workingDir string
		wantErr   bool
	}{
		{
			name:      "empty working dir",
			workingDir: "",
			wantErr:   false,
		},
		{
			name:      "base path",
			workingDir: basePath,
			wantErr:   false,
		},
		{
			name:      "subdirectory",
			workingDir: subPath,
			wantErr:   false,
		},
		{
			name:      "outside directory",
			workingDir: outsidePath,
			wantErr:   true,
		},
		{
			name:      "parent directory",
			workingDir: tempDir,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDir(basePath, tt.workingDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateWorkingDir(%q) expected error", tt.workingDir)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWorkingDir(%q) unexpected error: %v", tt.workingDir, err)
				}
			}
		})
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "normal path",
			s:    "path/to/file",
			want: false,
		},
		{
			name: "unix traversal",
			s:    "../../../etc/passwd",
			want: true,
		},
		{
			name: "windows traversal",
			s:    "..\\..\\windows\\system32",
			want: true,
		},
		{
			name: "mixed slashes",
			s:    "../path",
			want: true,
		},
		{
			name: "single dot",
			s:    "./file",
			want: false,
		},
		{
			name: "double dots in filename",
			s:    "file..name",
			want: false,
		},
		{
			name: "absolute path with traversal",
			s:    "/etc/../passwd",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsPathTraversal(tt.s)
			if got != tt.want {
				t.Errorf("ContainsPathTraversal(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
