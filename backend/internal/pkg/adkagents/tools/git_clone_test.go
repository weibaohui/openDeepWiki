package tools

import (
	"testing"
)

func TestGenerateRepoDirName(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    string
	}{
		{
			name:    "standard github url",
			repoURL: "https://github.com/user/repo.git",
			want:    "repo",
		},
		{
			name:    "url without .git suffix",
			repoURL: "https://github.com/user/repo",
			want:    "repo",
		},
		{
			name:    "url with path traversal",
			repoURL: "https://github.com/user/../etc/passwd",
			want:    "passwd",
		},
		{
			name:    "ssh url",
			repoURL: "git@github.com:user/repo.git",
			want:    "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRepoDirName(tt.repoURL)
			if got != tt.want {
				t.Errorf("GenerateRepoDirName(%q) = %q, want %q", tt.repoURL, got, tt.want)
			}
		})
	}
}

func TestContainsPathTraversal_InGitCloneContext(t *testing.T) {
	tests := []struct {
		name      string
		targetDir string
		want      bool
	}{
		{
			name:      "normal directory",
			targetDir: "myrepo",
			want:      false,
		},
		{
			name:      "directory with subdir",
			targetDir: "user/myrepo",
			want:      false,
		},
		{
			name:      "unix path traversal",
			targetDir: "../etc",
			want:      true,
		},
		{
			name:      "windows path traversal",
			targetDir: "..\\windows",
			want:      true,
		},
		{
			name:      "nested traversal",
			targetDir: "repo/../../../etc",
			want:      true,
		},
		{
			name:      "traversal at end",
			targetDir: "repo/..",
			want:      true,
		},
		{
			name:      "single dot",
			targetDir: "./repo",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsPathTraversal(tt.targetDir)
			if got != tt.want {
				t.Errorf("ContainsPathTraversal(%q) = %v, want %v", tt.targetDir, got, tt.want)
			}
		})
	}
}
