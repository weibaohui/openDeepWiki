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
	Dir           string `json:"dir"`
	Recursive     bool   `json:"recursive,omitempty"`
	Pattern       string `json:"pattern,omitempty"`
	IncludeConfig bool   `json:"include_config,omitempty"` // 默认 false，即默认忽略 .git, .idea, .vscode 等
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

	// 默认忽略的目录/文件
	ignoredNames := map[string]bool{
		".git":      true,
		".idea":     true,
		".vscode":   true,
		".DS_Store": true,
	}

	if params.Recursive {
		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 跳过无法访问的文件
			}

			// 跳过根目录本身
			if path == fullPath {
				return nil
			}

			// 检查是否需要忽略
			if !params.IncludeConfig && ignoredNames[d.Name()] {
				if d.IsDir() {
					return filepath.SkipDir
				}
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
			// 检查是否需要忽略
			if !params.IncludeConfig && ignoredNames[item.Name()] {
				continue
			}

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
