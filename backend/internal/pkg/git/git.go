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
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// CloneOptions 定义克隆仓库的参数。
type CloneOptions struct {
	URL       string
	TargetDir string
	Timeout   time.Duration
}

// FileChange 描述单个文件的变更信息。
type FileChange struct {
	Path        string
	ChangeType  string
	Description string
	LineRanges  []LineRange
}

// LineRange 描述文件变更的行号范围。
type LineRange struct {
	Start int
	End   int
	Side  string
}

// Clone 克隆远端仓库到指定目录。
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

// ParseRepoName 从仓库 URL 中解析仓库名。
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

// RemoveRepo 删除本地仓库目录。
func RemoveRepo(path string) error {
	return os.RemoveAll(path)
}

// IsValidGitURL 校验仓库 URL 格式是否合法。
func IsValidGitURL(url string) bool {
	httpsPattern := regexp.MustCompile(`^https?://[^\s]+\.git$|^https?://github\.com/[^\s]+$`)
	sshPattern := regexp.MustCompile(`^git@[^\s]+:[^\s]+\.git$`)

	return httpsPattern.MatchString(url) || sshPattern.MatchString(url)
}

// NormalizeRepoURL 归一化仓库 URL 并返回去重键。
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

// GetIncrementalChanges 获取指定提交到最新提交之间的文件变更与变更说明。
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
	lineRanges, err := getDiffLineRanges(repoPath, rangeExpr)
	if err != nil {
		return "", nil, err
	}

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
			LineRanges:  lineRanges[path],
		})
	}

	klog.V(6).Infof("增量变更统计完成: repoPath=%s, baseCommit=%s, latestCommit=%s, fileCount=%d", repoPath, baseCommit, latestCommit, len(result))

	return latestCommit, result, nil
}

// FormatIncrementalChangesForAI 格式化增量变更摘要，便于日志输出。
func FormatIncrementalChangesForAI(repoPath string, baseCommit string) (string, error) {
	latestCommit, changes, err := GetIncrementalChanges(repoPath, baseCommit)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("增量变更指引\n")
	builder.WriteString(fmt.Sprintf("最新提交: %s\n", latestCommit))
	builder.WriteString(fmt.Sprintf("变更文件数: %d\n", len(changes)))

	if len(changes) == 0 {
		builder.WriteString("结论: 未发现文件变更\n")
		return builder.String(), nil
	}

	counts := map[string]int{
		"新增":  0,
		"修改":  0,
		"删除":  0,
		"重命名": 0,
		"复制":  0,
		"未知":  0,
	}
	for _, change := range changes {
		if _, exists := counts[change.ChangeType]; !exists {
			counts[change.ChangeType] = 0
		}
		counts[change.ChangeType]++
	}

	fmt.Fprintf(&builder, "变更类型统计: 新增 %d, 修改 %d, 删除 %d, 重命名 %d, 复制 %d, 未知 %d\n",
		counts["新增"], counts["修改"], counts["删除"], counts["重命名"], counts["复制"], counts["未知"])

	fmt.Fprintf(&builder, "文件清单:\n")
	for index, change := range changes {
		fmt.Fprintf(&builder, "%d. %s | %s\n", index+1, change.Path, change.ChangeType)
		if change.Description != "" {
			fmt.Fprintf(&builder, "   说明: %s\n", change.Description)
		}
		if len(change.LineRanges) == 0 {
			fmt.Fprintf(&builder, "   行号: 未提供\n")
		} else {
			fmt.Fprintf(&builder, "   行号: %s\n", formatLineRanges(change.LineRanges))
		}
		fmt.Fprintf(&builder, "   建议: %s\n", buildChangeSuggestion(change))
	}

	return builder.String(), nil
}

// Pull 执行 git pull 并返回输出。
func Pull(ctx context.Context, repoPath string) (string, error) {
	return runGitCommand(ctx, repoPath, "pull")
}

// EnsureBaseCommitAvailable 确保基线提交在仓库历史中可用，必要时补全历史记录。
func EnsureBaseCommitAvailable(ctx context.Context, repoPath string, baseCommit string) error {
	baseCommit = strings.TrimSpace(baseCommit)
	if baseCommit == "" {
		return fmt.Errorf("仓库基线提交为空")
	}

	verifyErr := verifyCommit(ctx, repoPath, baseCommit)
	if verifyErr == nil {
		return nil
	}

	isShallow, shallowErr := isShallowRepository(ctx, repoPath)
	if shallowErr != nil {
		return fmt.Errorf("基线提交校验失败: %w", verifyErr)
	}
	if isShallow {
		klog.V(6).Infof("仓库为浅克隆，尝试补全历史: repoPath=%s", repoPath)
		if err := fetchUnshallow(ctx, repoPath); err != nil {
			klog.V(6).Infof("补全历史失败: repoPath=%s, error=%v", repoPath, err)
		} else if ok, err := tryFetchAllAndVerify(ctx, repoPath, baseCommit); err == nil && ok {
			return nil
		}
	}

	if ok, err := tryFetchAllAndVerify(ctx, repoPath, baseCommit); err == nil && ok {
		return nil
	}

	return fmt.Errorf("基线提交不可用，请检查仓库历史或更新基线提交: %w", verifyErr)
}

// tryFetchAllAndVerify 拉取远端历史并校验基线提交是否存在。
func tryFetchAllAndVerify(ctx context.Context, repoPath string, baseCommit string) (bool, error) {
	if err := ensureAllBranches(ctx, repoPath); err != nil {
		klog.V(6).Infof("设置远端分支拉取失败: repoPath=%s, error=%v", repoPath, err)
	}
	if err := fetchAll(ctx, repoPath); err != nil {
		return false, err
	}
	if err := verifyCommit(ctx, repoPath, baseCommit); err != nil {
		return false, nil
	}
	return true, nil
}

func formatLineRanges(ranges []LineRange) string {
	parts := make([]string, 0, len(ranges))
	for _, item := range ranges {
		side := "未知"
		if item.Side == "new" {
			side = "新版"
		} else if item.Side == "old" {
			side = "旧版"
		}
		if item.Start == item.End {
			parts = append(parts, fmt.Sprintf("%s %d", side, item.Start))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d-%d", side, item.Start, item.End))
	}
	return strings.Join(parts, "；")
}

func buildChangeSuggestion(change FileChange) string {
	switch change.ChangeType {
	case "新增":
		return "优先阅读新增内容的用途与依赖关系"
	case "修改":
		return "重点关注修改行号及其上下游调用"
	case "删除":
		return "确认删除原因及是否存在替代实现"
	case "重命名":
		return "关注命名调整是否影响引用路径"
	case "复制":
		return "确认复制的用途是否引入重复逻辑"
	default:
		if len(change.LineRanges) == 0 {
			return "需要人工确认变更细节"
		}
		return "结合行号范围进行快速复核"
	}
}

// getDiffLineRanges 获取指定提交范围内的行号区间
func getDiffLineRanges(repoPath string, rangeExpr string) (map[string][]LineRange, error) {
	cmd := exec.Command("git", "diff", "--unified=0", "--no-color", rangeExpr)
	cmd.Dir = repoPath
	diffBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w, output: %s", err, string(diffBytes))
	}

	output := strings.TrimSpace(string(diffBytes))
	if output == "" {
		return map[string][]LineRange{}, nil
	}

	ranges := make(map[string][]LineRange)
	var currentPath string
	hunkRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				path := parts[3]
				currentPath = strings.TrimPrefix(path, "b/")
			}
			continue
		}

		if strings.HasPrefix(line, "rename to ") {
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "rename to "))
			continue
		}

		if !strings.HasPrefix(line, "@@ ") {
			continue
		}

		if currentPath == "" {
			continue
		}

		matches := hunkRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}

		oldStart := parseInt(matches[1])
		oldCount := parseIntWithDefault(matches[2], 1)
		newStart := parseInt(matches[3])
		newCount := parseIntWithDefault(matches[4], 1)

		if newCount == 0 && oldCount == 0 {
			continue
		}

		var start int
		var count int
		side := "new"
		if newCount == 0 && oldCount > 0 {
			start = oldStart
			count = oldCount
			side = "old"
		} else {
			start = newStart
			count = newCount
		}

		if count <= 0 {
			continue
		}

		end := start + count - 1
		ranges[currentPath] = append(ranges[currentPath], LineRange{
			Start: start,
			End:   end,
			Side:  side,
		})
	}

	return ranges, nil
}

// parseInt 解析整数
func parseInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

// parseIntWithDefault 解析整数并提供默认值
func parseIntWithDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// GetBranchAndCommit 获取当前分支名与最新提交号。
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

func verifyCommit(ctx context.Context, repoPath string, commit string) error {
	_, err := runGitCommand(ctx, repoPath, "rev-parse", "--verify", commit)
	return err
}

func isShallowRepository(ctx context.Context, repoPath string) (bool, error) {
	output, err := runGitCommand(ctx, repoPath, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(output), "true"), nil
}

func fetchUnshallow(ctx context.Context, repoPath string) error {
	_, err := runGitCommand(ctx, repoPath, "fetch", "--unshallow", "--tags", "--prune", "--force")
	return err
}

func fetchAll(ctx context.Context, repoPath string) error {
	_, err := runGitCommand(ctx, repoPath, "fetch", "--all", "--tags", "--prune", "--force")
	return err
}

func ensureAllBranches(ctx context.Context, repoPath string) error {
	_, err := runGitCommand(ctx, repoPath, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	return err
}

func runGitCommand(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s 失败: %w, 输出: %s", strings.Join(args, " "), err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// DirSizeMB 统计目录大小（MB）。
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
