package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CountLinesArgs count_lines 工具参数
type CountLinesArgs struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

// CountLines 统计文件行数
func CountLines(args json.RawMessage, basePath string) (string, error) {
	var params CountLinesArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if params.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// 检查路径参数是否为绝对路径
	// TODO 改为验证是否在项目的仓库范围内
	// if filepath.IsAbs(params.Path) {
	// 	return "", fmt.Errorf("absolute paths not allowed: %s", params.Path)
	// }

	// 构建完整路径
	fullPath := params.Path

	// 安全检查
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", params.Path)
	}

	// 获取文件信息
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path not found: %s", params.Path)
		}
		return "", fmt.Errorf("cannot access path: %w", err)
	}

	var totalLines int64
	var fileCount int

	if info.IsDir() {
		// 目录：递归统计
		totalLines, fileCount, err = countLinesInDir(fullPath, params.Pattern)
		if err != nil {
			return "", err
		}
	} else {
		// 单个文件
		lines, err := countLinesInFile(fullPath)
		if err != nil {
			return "", err
		}
		totalLines = lines
		fileCount = 1
	}

	// 格式化输出
	var output strings.Builder
	if info.IsDir() {
		if params.Pattern != "" {
			output.WriteString(fmt.Sprintf("Directory: %s (pattern: %s)\n", params.Path, params.Pattern))
		} else {
			output.WriteString(fmt.Sprintf("Directory: %s\n", params.Path))
		}
		output.WriteString(fmt.Sprintf("Files counted: %d\n", fileCount))
		output.WriteString(fmt.Sprintf("Total lines: %d", totalLines))
	} else {
		output.WriteString(fmt.Sprintf("File: %s\n", params.Path))
		output.WriteString(fmt.Sprintf("Lines: %d", totalLines))
	}

	return output.String(), nil
}

// countLinesInDir 统计目录中的总行数
func countLinesInDir(root string, pattern string) (int64, int, error) {
	var totalLines int64
	var fileCount int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过无法访问的文件
		}

		if d.IsDir() {
			return nil
		}

		// 应用 glob 过滤
		if pattern != "" {
			matched, _ := filepath.Match(pattern, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// 跳过二进制文件和过大的文件
		info, err := d.Info()
		if err != nil || info.Size() > 10*1024*1024 { // 跳过 > 10MB 的文件
			return nil
		}

		lines, err := countLinesInFile(path)
		if err != nil {
			return nil // 跳过无法读取的文件
		}

		totalLines += lines
		fileCount++

		// 限制文件数量
		if fileCount > 10000 {
			return fmt.Errorf("too many files")
		}

		return nil
	})

	return totalLines, fileCount, err
}

// countLinesInFile 统计单个文件的行数
func countLinesInFile(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines int64

	// 增加缓冲区大小以处理长行
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		lines++
	}

	return lines, scanner.Err()
}
