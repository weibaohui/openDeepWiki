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

	"k8s.io/klog/v2"
)

type CloneOptions struct {
	URL       string
	TargetDir string
	Timeout   time.Duration
}

type FileChange struct {
	Path        string
	ChangeType  string
	Description string
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

// GetIncrementalChanges 获取指定提交到最新提交之间的文件变更与变更说明
func GetIncrementalChanges(repoPath string, baseCommit string) (string, []FileChange, error) {
	baseCommit = strings.TrimSpace(baseCommit)
	if baseCommit == "" {
		return "", nil, fmt.Errorf("base commit is empty")
	}

	klog.V(6).Infof("开始获取增量变更: repoPath=%s, baseCommit=%s", repoPath, baseCommit)

	baseVerifyCmd := exec.Command("git", "rev-parse", "--verify", baseCommit)
	baseVerifyCmd.Dir = repoPath
	baseBytes, err := baseVerifyCmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("base commit not found: %w, output: %s", err, string(baseBytes))
	}
	baseResolved := strings.TrimSpace(string(baseBytes))

	headVerifyCmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	headVerifyCmd.Dir = repoPath
	headBytes, err := headVerifyCmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("head commit not found: %w, output: %s", err, string(headBytes))
	}
	headResolved := strings.TrimSpace(string(headBytes))

	latestCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	latestCmd.Dir = repoPath
	latestBytes, err := latestCmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("git latest commit failed: %w, output: %s", err, string(latestBytes))
	}
	latestCommit := strings.TrimSpace(string(latestBytes))

	if baseResolved == headResolved {
		klog.V(6).Infof("增量变更为空: repoPath=%s, baseCommit=%s", repoPath, baseCommit)
		return latestCommit, []FileChange{}, nil
	}

	rangeExpr := fmt.Sprintf("%s..HEAD", baseResolved)
	logCmd := exec.Command("git", "log", "--name-status", "--pretty=format:@@@%H|%s", rangeExpr)
	logCmd.Dir = repoPath
	logBytes, err := logCmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("git log failed: %w, output: %s", err, string(logBytes))
	}

	output := strings.TrimSpace(string(logBytes))
	if output == "" {
		return latestCommit, []FileChange{}, nil
	}

	type changeItem struct {
		Path         string
		ChangeType   string
		Descriptions []string
	}

	changes := make(map[string]*changeItem)
	order := make([]string, 0)
	var currentSubject string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "@@@") {
			header := strings.TrimPrefix(line, "@@@")
			parts := strings.SplitN(header, "|", 2)
			if len(parts) == 2 {
				currentSubject = strings.TrimSpace(parts[1])
			} else {
				currentSubject = ""
			}
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := strings.TrimSpace(fields[0])
		path := strings.TrimSpace(fields[1])
		description := currentSubject

		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if len(fields) >= 3 {
				oldPath := strings.TrimSpace(fields[1])
				newPath := strings.TrimSpace(fields[2])
				path = newPath
				if description != "" {
					description = fmt.Sprintf("%s（重命名: %s -> %s）", description, oldPath, newPath)
				} else {
					description = fmt.Sprintf("重命名: %s -> %s", oldPath, newPath)
				}
			}
		}

		changeType := "未知"
		switch {
		case strings.HasPrefix(status, "A"):
			changeType = "新增"
		case strings.HasPrefix(status, "M"):
			changeType = "修改"
		case strings.HasPrefix(status, "D"):
			changeType = "删除"
		case strings.HasPrefix(status, "R"):
			changeType = "重命名"
		case strings.HasPrefix(status, "C"):
			changeType = "复制"
		}

		item, exists := changes[path]
		if !exists {
			item = &changeItem{
				Path:         path,
				ChangeType:   changeType,
				Descriptions: make([]string, 0),
			}
			changes[path] = item
			order = append(order, path)
		}
		if item.ChangeType == "" {
			item.ChangeType = changeType
		}

		if description != "" {
			duplicated := false
			for _, existing := range item.Descriptions {
				if existing == description {
					duplicated = true
					break
				}
			}
			if !duplicated {
				item.Descriptions = append(item.Descriptions, description)
			}
		}
	}

	result := make([]FileChange, 0, len(order))
	for _, path := range order {
		item := changes[path]
		result = append(result, FileChange{
			Path:        item.Path,
			ChangeType:  item.ChangeType,
			Description: strings.Join(item.Descriptions, "；"),
		})
	}

	klog.V(6).Infof("增量变更统计完成: repoPath=%s, baseCommit=%s, latestCommit=%s, fileCount=%d", repoPath, baseCommit, latestCommit, len(result))

	return latestCommit, result, nil
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
