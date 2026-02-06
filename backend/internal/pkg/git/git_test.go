package git

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v error: %v, output=%s", args, err, string(output))
	}
}

func TestDirSizeMB(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 2*1024*1024)
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), data, 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	size, err := DirSizeMB(dir)
	if err != nil {
		t.Fatalf("DirSizeMB error: %v", err)
	}
	if math.Abs(size-2.0) > 0.05 {
		t.Fatalf("unexpected sizeMB: %.4f", size)
	}
}

func TestGetBranchAndCommit(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		runGit(t, dir, "init")
	} else if len(output) == 0 {
	}

	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")

	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")

	branch, commit, err := GetBranchAndCommit(dir)
	if err != nil {
		t.Fatalf("GetBranchAndCommit error: %v", err)
	}
	if branch == "" {
		t.Fatalf("branch is empty")
	}
	if commit == "" {
		t.Fatalf("commit is empty")
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantURL   string
		wantKey   string
		expectErr bool
	}{
		{
			name:      "https with query",
			input:     "https://github.com/Owner/Repo?tab=readme",
			wantURL:   "https://github.com/owner/repo",
			wantKey:   "github.com/owner/repo",
			expectErr: false,
		},
		{
			name:      "https with git suffix",
			input:     "https://gitlab.com/team/project.git",
			wantURL:   "https://gitlab.com/team/project",
			wantKey:   "gitlab.com/team/project",
			expectErr: false,
		},
		{
			name:      "ssh url",
			input:     "git@github.com:Owner/Repo.git",
			wantURL:   "git@github.com:owner/repo.git",
			wantKey:   "github.com/owner/repo",
			expectErr: false,
		},
		{
			name:      "invalid path depth",
			input:     "https://github.com/owner/repo/blob/main/README.md",
			expectErr: true,
		},
		{
			name:      "invalid scheme",
			input:     "ftp://github.com/owner/repo",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotKey, err := NormalizeRepoURL(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeRepoURL error: %v", err)
			}
			if gotURL != tt.wantURL {
				t.Fatalf("unexpected url: %s", gotURL)
			}
			if gotKey != tt.wantKey {
				t.Fatalf("unexpected key: %s", gotKey)
			}
		})
	}
}
