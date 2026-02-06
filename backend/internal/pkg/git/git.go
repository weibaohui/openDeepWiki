package git

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
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

func NormalizeRepoURL(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("empty url")
	}

	if strings.HasPrefix(trimmed, "git@") {
		re := regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?/?$`)
		matches := re.FindStringSubmatch(trimmed)
		if len(matches) != 4 {
			return "", "", fmt.Errorf("invalid ssh url")
		}
		host := strings.ToLower(matches[1])
		owner := strings.ToLower(matches[2])
		repo := strings.ToLower(matches[3])
		normalized := fmt.Sprintf("git@%s:%s/%s.git", host, owner, repo)
		key := fmt.Sprintf("%s/%s/%s", host, owner, repo)
		return normalized, key, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("unsupported scheme")
	}
	if parsed.Host == "" {
		return "", "", fmt.Errorf("missing host")
	}

	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	if path == "" {
		return "", "", fmt.Errorf("missing repo path")
	}
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo path")
	}
	host := strings.ToLower(parsed.Host)
	owner := strings.ToLower(parts[0])
	repo := strings.ToLower(parts[1])
	normalized := fmt.Sprintf("%s://%s/%s/%s", strings.ToLower(parsed.Scheme), host, owner, repo)
	key := fmt.Sprintf("%s/%s/%s", host, owner, repo)
	return normalized, key, nil
}

func GetBranchAndCommit(repoPath string) (string, string, error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoPath
	branchBytes, err := branchCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git branch failed: %w, output: %s", err, string(branchBytes))
	}

	commitCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir = repoPath
	commitBytes, err := commitCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("git commit failed: %w, output: %s", err, string(commitBytes))
	}

	return strings.TrimSpace(string(branchBytes)), strings.TrimSpace(string(commitBytes)), nil
}

func DirSizeMB(path string) (float64, error) {
	var totalSize int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}

	return float64(totalSize) / (1024 * 1024), nil
}
