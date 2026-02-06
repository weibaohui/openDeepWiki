package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type CloneOptions struct {
	URL       string
	TargetDir string
	Timeout   time.Duration
}

func Clone(opts CloneOptions) error {
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	url := opts.URL

	if err := os.MkdirAll(filepath.Dir(opts.TargetDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, opts.TargetDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s, output: %s", err, string(output))
	}

	return nil
}

func ParseRepoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	re := regexp.MustCompile(`[:/]([^/:]+/[^/:]+)$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		parts := strings.Split(matches[1], "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}

	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func RemoveRepo(path string) error {
	return os.RemoveAll(path)
}

func IsValidGitURL(url string) bool {
	httpsPattern := regexp.MustCompile(`^https?://[^\s]+\.git$|^https?://github\.com/[^\s]+$`)
	sshPattern := regexp.MustCompile(`^git@[^\s]+:[^\s]+\.git$`)

	return httpsPattern.MatchString(url) || sshPattern.MatchString(url)
}
