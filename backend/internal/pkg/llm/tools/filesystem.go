// Filesystem Tools - 文件系统操作工具
// 对应 MCP tools: filesystem.ls, filesystem.stat, filesystem.exists, filesystem.find
// 扩展现有功能: filesystem.read (已有), filesystem.grep (通过 search_text 已有)

package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ListDirArgs filesystem.ls 参数
type ListDirArgs struct {
	Dir       string `json:"dir"`
	Recursive bool   `json:"recursive,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
}

// ListDirEntry 目录条目
type ListDirEntry struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Size     int64     `json:"size,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

// ListDir 列出目录内容
func ListDir(args json.RawMessage, basePath string) (string, error) {
	var params ListDirArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Dir == "" {
		params.Dir = "."
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.Dir)
	if strings.HasPrefix(params.Dir, "/") {
		fullPath = params.Dir
	}
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("dir escapes base directory: %s", params.Dir)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", params.Dir)
		}
		return "", fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", params.Dir)
	}

	var entries []ListDirEntry

	if params.Recursive {
		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 跳过无法访问的文件
			}

			// 跳过根目录本身
			if path == fullPath {
				return nil
			}

			relPath, _ := filepath.Rel(fullPath, path)

			// 如果指定了 pattern，进行过滤
			if params.Pattern != "" {
				matched, _ := filepath.Match(params.Pattern, d.Name())
				if !matched && !d.IsDir() {
					return nil
				}
			}

			entry := ListDirEntry{
				Name: relPath,
				Type: "file",
			}
			if d.IsDir() {
				entry.Type = "directory"
			}

			if info, err := d.Info(); err == nil {
				entry.Size = info.Size()
				entry.Modified = info.ModTime()
			}

			entries = append(entries, entry)
			return nil
		})
	} else {
		items, err := os.ReadDir(fullPath)
		if err != nil {
			return "", fmt.Errorf("cannot read directory: %w", err)
		}

		for _, item := range items {
			// 如果指定了 pattern，进行过滤
			if params.Pattern != "" {
				matched, _ := filepath.Match(params.Pattern, item.Name())
				if !matched && !item.IsDir() {
					continue
				}
			}

			entry := ListDirEntry{
				Name: item.Name(),
				Type: "file",
			}
			if item.IsDir() {
				entry.Type = "directory"
			}

			if info, err := item.Info(); err == nil {
				entry.Size = info.Size()
				entry.Modified = info.ModTime()
			}

			entries = append(entries, entry)
		}
	}

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	// 格式化输出
	var lines []string
	for _, e := range entries {
		typeStr := "F"
		if e.Type == "directory" {
			typeStr = "D"
		}
		lines = append(lines, fmt.Sprintf("[%s] %-50s %10d %s",
			typeStr, e.Name, e.Size, e.Modified.Format("2006-01-02 15:04")))
	}

	return strings.Join(lines, "\n"), nil
}

// FileStatArgs filesystem.stat 参数
type FileStatArgs struct {
	FilePath string `json:"file_path"`
}

// FileStat 获取文件元信息
func FileStat(args json.RawMessage, basePath string) (string, error) {
	var params FileStatArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.FilePath)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("file_path escapes base directory: %s", params.FilePath)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", params.FilePath)
		}
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	fileType := "file"
	if info.IsDir() {
		fileType = "directory"
	}

	result := fmt.Sprintf(`Name: %s
Type: %s
Size: %d bytes
Modified: %s
Created: %s (platform dependent)
Permissions: %s`,
		info.Name(),
		fileType,
		info.Size(),
		info.ModTime().Format("2006-01-02 15:04:05 MST"),
		info.ModTime().Format("2006-01-02 15:04:05 MST"), // Note: Creation time not available on all platforms
		info.Mode().String(),
	)

	return result, nil
}

// FileExistsArgs filesystem.exists 参数
type FileExistsArgs struct {
	Path string `json:"path"`
}

// FileExists 检查文件或目录是否存在
func FileExistsTool(args json.RawMessage, basePath string) (string, error) {
	var params FileExistsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.Path)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("exists: false\ntype: none"), nil
		}
		return "", fmt.Errorf("cannot access path: %w", err)
	}

	fileType := "file"
	if info.IsDir() {
		fileType = "directory"
	}

	return fmt.Sprintf("exists: true\ntype: %s", fileType), nil
}

// FindFilesArgs filesystem.find 参数
type FindFilesArgs struct {
	Path        string `json:"path"`
	NamePattern string `json:"name_pattern,omitempty"`
	Type        string `json:"type,omitempty"` // "file" or "directory"
	MaxDepth    int    `json:"max_depth,omitempty"`
}

// FindFiles 查找文件
func FindFiles(args json.RawMessage, basePath string) (string, error) {
	var params FindFilesArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Path == "" {
		params.Path = "."
	}

	// 安全检查
	fullPath := filepath.Join(basePath, params.Path)
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	if params.MaxDepth <= 0 {
		params.MaxDepth = 10
	}
	if params.MaxDepth > 20 {
		params.MaxDepth = 20 // 限制最大深度
	}

	var results []string
	// currentDepth := 0

	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}

		// 计算深度
		if path == fullPath {
			return nil
		}
		rel, _ := filepath.Rel(fullPath, path)
		depth := strings.Count(rel, string(filepath.Separator)) + 1

		if depth > params.MaxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// 类型过滤
		if params.Type != "" {
			isDir := d.IsDir()
			if params.Type == "directory" && !isDir {
				return nil
			}
			if params.Type == "file" && isDir {
				return nil
			}
		}

		// 名称模式过滤
		if params.NamePattern != "" {
			matched, _ := filepath.Match(params.NamePattern, d.Name())
			if !matched {
				return nil
			}
		}

		// 添加到结果
		relPath, _ := filepath.Rel(basePath, path)
		results = append(results, relPath)

		// 限制结果数量
		if len(results) >= 100 {
			return fmt.Errorf("max results reached")
		}

		return nil
	})

	if err != nil && err.Error() != "max results reached" {
		return "", fmt.Errorf("error searching files: %w", err)
	}

	if len(results) == 0 {
		return "No files found matching the criteria.", nil
	}

	result := fmt.Sprintf("Found %d items:\n%s", len(results), strings.Join(results, "\n"))
	if err != nil && err.Error() == "max results reached" {
		result += "\n... (results truncated)"
	}

	return result, nil
}
