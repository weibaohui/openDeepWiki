package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitCloneArgs struct {
	RepoURL   string `json:"repo_url"`
	Branch    string `json:"branch,omitempty"`
	TargetDir string `json:"target_dir"`
	Depth     int    `json:"depth,omitempty"`
}

func GitClone(args json.RawMessage, basePath string) (string, error) {
	var params GitCloneArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.RepoURL == "" {
		return "", fmt.Errorf("repo_url is required")
	}
	if params.TargetDir == "" {
		return "", fmt.Errorf("target_dir is required")
	}

	targetPath := filepath.Join(basePath, params.TargetDir)
	if !isPathSafe(basePath, targetPath) {
		return "", fmt.Errorf("target_dir escapes base directory: %s", params.TargetDir)
	}

	cmdArgs := []string{"clone"}

	if params.Depth > 0 {
		cmdArgs = append(cmdArgs, "--depth", fmt.Sprintf("%d", params.Depth))
	}

	if params.Branch != "" {
		cmdArgs = append(cmdArgs, "-b", params.Branch)
	}

	cmdArgs = append(cmdArgs, params.RepoURL, targetPath)

	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	branch, commit, _ := getGitInfo(targetPath)
	if branch == "" {
		branch = params.Branch
		if branch == "" {
			branch = "main"
		}
	}

	result := fmt.Sprintf("Successfully cloned repository to %s\nBranch: %s\nCommit: %s",
		params.TargetDir, branch, commit)
	return result, nil
}

type GitDiffArgs struct {
	CommitHash string `json:"commit_hash"`
	FilePath   string `json:"file_path,omitempty"`
}

func GitDiff(args json.RawMessage, basePath string) (string, error) {
	var params GitDiffArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.CommitHash == "" {
		return "", fmt.Errorf("commit_hash is required")
	}

	cmdArgs := []string{"diff", params.CommitHash}
	if params.FilePath != "" {
		fullPath := filepath.Join(basePath, params.FilePath)
		if !isPathSafe(basePath, fullPath) {
			return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
		}
		cmdArgs = append(cmdArgs, "--", fullPath)
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = basePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	if len(output) == 0 {
		return "No differences found.", nil
	}

	return string(output), nil
}

type GitLogArgs struct {
	FilePath string `json:"file_path,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Since    string `json:"since,omitempty"`
}

func GitLog(args json.RawMessage, basePath string) (string, error) {
	var params GitLogArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 50 {
		params.Limit = 50
	}

	cmdArgs := []string{"log", "--oneline", fmt.Sprintf("-%d", params.Limit)}

	if params.Since != "" {
		cmdArgs = append(cmdArgs, "--since", params.Since)
	}

	if params.FilePath != "" {
		fullPath := filepath.Join(basePath, params.FilePath)
		if !isPathSafe(basePath, fullPath) {
			return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
		}
		cmdArgs = append(cmdArgs, "--", fullPath)
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = basePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	return string(output), nil
}

type GitStatusArgs struct {
	RepoPath string `json:"repo_path"`
}

func GitStatus(args json.RawMessage, basePath string) (string, error) {
	var params GitStatusArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	repoPath := basePath
	if params.RepoPath != "" {
		repoPath = filepath.Join(basePath, params.RepoPath)
		if !isPathSafe(basePath, repoPath) {
			return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
		}
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoPath
	branch, _ := branchCmd.Output()

	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = repoPath
	status, _ := statusCmd.Output()

	isClean := len(strings.TrimSpace(string(status))) == 0

	var modified, untracked []string
	for _, line := range strings.Split(string(status), "\n") {
		if len(line) < 3 {
			continue
		}
		statusCode := line[:2]
		file := strings.TrimSpace(line[2:])

		if strings.Contains(statusCode, "?") {
			untracked = append(untracked, file)
		} else {
			modified = append(modified, file)
		}
	}

	result := fmt.Sprintf("Branch: %s\nClean: %v\n", strings.TrimSpace(string(branch)), isClean)
	if len(modified) > 0 {
		result += fmt.Sprintf("Modified (%d):\n%s\n", len(modified), strings.Join(modified, "\n"))
	}
	if len(untracked) > 0 {
		result += fmt.Sprintf("Untracked (%d):\n%s\n", len(untracked), strings.Join(untracked, "\n"))
	}

	return result, nil
}

type GitBranchListArgs struct {
	RepoPath string `json:"repo_path"`
	Remote   bool   `json:"remote,omitempty"`
}

func GitBranchList(args json.RawMessage, basePath string) (string, error) {
	var params GitBranchListArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	repoPath := basePath
	if params.RepoPath != "" {
		repoPath = filepath.Join(basePath, params.RepoPath)
		if !isPathSafe(basePath, repoPath) {
			return "", fmt.Errorf("repo_path escapes base directory: %s", params.RepoPath)
		}
	}

	cmdArgs := []string{"branch"}
	if params.Remote {
		cmdArgs = append(cmdArgs, "-r")
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git branch failed: %w", err)
	}

	currentCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentCmd.Dir = repoPath
	current, _ := currentCmd.Output()

	result := fmt.Sprintf("Current: %s\n\n%s", strings.TrimSpace(string(current)), string(output))
	return result, nil
}

func getGitInfo(repoPath string) (branch, commit string, err error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoPath
	branchBytes, _ := branchCmd.Output()
	branch = strings.TrimSpace(string(branchBytes))

	commitCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir = repoPath
	commitBytes, _ := commitCmd.Output()
	commit = strings.TrimSpace(string(commitBytes))

	return branch, commit, nil
}
