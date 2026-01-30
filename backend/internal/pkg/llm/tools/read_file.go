package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileArgs read_file 工具参数
type ReadFileArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// ReadFile 读取文件内容
func ReadFile(args json.RawMessage, basePath string) (string, error) {
	var params ReadFileArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// // 检查路径参数是否为绝对路径
	// TODO 改为验证是否在项目的仓库范围内
	// if filepath.IsAbs(params.Path) {
	// 	return "", fmt.Errorf("absolute paths not allowed: %s", params.Path)
	// }

	// 构建完整路径
	fullPath := filepath.Join(basePath, params.Path)

	// 安全检查
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", params.Path)
		}
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", params.Path)
	}

	// 检查文件大小限制 (1MB)
	const maxFileSize = 1024 * 1024
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file too large (max 1MB): %s (%d bytes)", params.Path, info.Size())
	}

	// 设置默认值
	offset := params.Offset
	if offset < 1 {
		offset = 1
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500 // 最大 500 行
	}

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// 按行读取
	scanner := bufio.NewScanner(file)
	var lines []string
	currentLine := 0

	for scanner.Scan() {
		currentLine++
		if currentLine < offset {
			continue
		}
		if currentLine >= offset+limit {
			lines = append(lines, fmt.Sprintf("... (%d more lines)", int(info.Size())/100)) // 粗略估计
			break
		}
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no content at offset %d", offset)
	}

	return strings.Join(lines, "\n"), nil
}
