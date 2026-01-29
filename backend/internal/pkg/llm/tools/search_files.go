package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SearchFilesArgs search_files 工具参数
type SearchFilesArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// SearchFiles 搜索匹配模式的文件
func SearchFiles(args json.RawMessage, basePath string) (string, error) {
	var params SearchFilesArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// 检查路径参数是否为绝对路径（应该使用相对路径）
	if filepath.IsAbs(params.Path) {
		return "", fmt.Errorf("absolute paths not allowed: %s", params.Path)
	}

	// 使用 basePath 或 params.Path 作为搜索路径
	searchPath := basePath
	if params.Path != "" {
		searchPath = filepath.Join(basePath, params.Path)
	}

	// 安全检查：确保搜索路径在 basePath 内
	if !isPathSafe(basePath, searchPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	// 执行搜索
	files, err := globSearch(searchPath, params.Pattern)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// 限制结果数量
	const maxResults = 100
	if len(files) > maxResults {
		files = files[:maxResults]
		files = append(files, fmt.Sprintf("... (%d more results truncated)", len(files)-maxResults))
	}

	if len(files) == 0 {
		return "No files found matching the pattern.", nil
	}

	return strings.Join(files, "\n"), nil
}

// globSearch 执行 glob 搜索，支持 ** 递归匹配
func globSearch(root, pattern string) ([]string, error) {
	var results []string

	// 如果模式不包含 **，使用标准库的 Glob
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			rel, _ := filepath.Rel(root, m)
			results = append(results, rel)
		}
		return results, nil
	}

	// 处理 ** 递归匹配
	// 将 ** 替换为通配符遍历
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid pattern with multiple **: %s", pattern)
	}

	prefix := strings.TrimPrefix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}

		if d.IsDir() {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		// 检查是否匹配前缀
		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		// 检查后缀
		if suffix != "" {
			// 将后缀的 glob 转换为匹配
			matched, err := filepath.Match(suffix, filepath.Base(relPath))
			if err != nil || !matched {
				return nil
			}
		}

		results = append(results, relPath)
		return nil
	})

	return results, err
}

// isPathSafe 检查路径是否在 basePath 内
func isPathSafe(basePath, targetPath string) bool {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	// 清理路径
	absBase = filepath.Clean(absBase)
	absTarget = filepath.Clean(absTarget)

	// 确保目标路径以基础路径为前缀
	if absTarget == absBase {
		return true
	}
	// 添加分隔符确保是完整的前缀匹配（防止 /foo/bar 匹配 /foo/barbaz）
	return strings.HasPrefix(absTarget, absBase+string(filepath.Separator))
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
