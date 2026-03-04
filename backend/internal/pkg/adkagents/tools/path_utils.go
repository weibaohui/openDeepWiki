package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isPathSafe 检查目标路径是否在基础目录范围内
// 增强版本：解析符号链接，防止路径遍历攻击
func isPathSafe(basePath, targetPath string) bool {
	// 1. 获取基础目录的绝对路径
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}

	// 2. 处理目标路径
	// 如果是相对路径，先与基础目录连接
	var target string
	if filepath.IsAbs(targetPath) {
		target = targetPath
	} else {
		target = filepath.Join(absBase, targetPath)
	}

	// 3. 获取目标路径的绝对路径
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	// 4. 清理路径（移除 . 和 ..）
	absBase = filepath.Clean(absBase)
	absTarget = filepath.Clean(absTarget)

	// 5. 解析符号链接（关键安全改进）
	// 注意：EvalSymlinks 要求路径存在，对于不存在的路径，我们尝试解析其父目录
	realBase := resolvePathWithSymlinks(absBase)
	realTarget := resolvePathWithSymlinks(absTarget)

	// 6. 检查目标路径是否等于基础目录
	if realTarget == realBase {
		return true
	}

	// 7. 确保基础目录以分隔符结尾，避免前缀匹配问题
	// 例如：/safe/base 和 /safe/base_backup 不应该被认为是包含关系
	prefix := realBase
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}

	// 8. 使用分隔符确保精确匹配目录边界
	return strings.HasPrefix(realTarget+string(filepath.Separator), prefix)
}

// resolvePathWithSymlinks 解析路径中的符号链接
// 如果路径不存在，尝试解析其存在的父目录
func resolvePathWithSymlinks(path string) string {
	// 首先尝试直接解析
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved)
	}

	// 如果路径不存在，逐步向上查找存在的目录并解析
	// 然后拼接剩余部分
	dir := path
	var unresolvedParts []string

	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// 到达根目录，无法继续
			break
		}

		// 记录当前目录名（未解析的部分）
		unresolvedParts = append([]string{filepath.Base(dir)}, unresolvedParts...)

		// 尝试解析父目录
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err == nil {
			// 找到了存在的、可解析的父目录
			// 将解析后的父目录与未解析的部分拼接
			result := resolvedParent
			for _, part := range unresolvedParts[1:] { // 跳过第一个（已解析的父目录本身）
				result = filepath.Join(result, part)
			}
			return filepath.Clean(result)
		}

		// 父目录也不存在，继续向上
		dir = parent
	}

	// 无法解析任何符号链接，返回原始清理后的路径
	return filepath.Clean(path)
}

// ValidateAndResolvePath 验证并解析目标路径
// 返回解析后的绝对路径，如果路径不安全则返回错误
func ValidateAndResolvePath(basePath, inputPath string) (string, error) {
	// 构建完整路径
	fullPath := filepath.Join(basePath, inputPath)
	if filepath.IsAbs(inputPath) {
		fullPath = inputPath
	}

	// 验证路径安全
	if !isPathSafe(basePath, fullPath) {
		return "", fmt.Errorf("path escapes base directory: %s", inputPath)
	}

	// 解析最终路径
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	return absPath, nil
}

// ValidateWorkingDir 验证工作目录是否在基础目录范围内
func ValidateWorkingDir(basePath, workingDir string) error {
	if workingDir == "" {
		return nil
	}
	if !isPathSafe(basePath, workingDir) {
		return fmt.Errorf("working directory escapes base directory: %s", workingDir)
	}
	return nil
}

// ContainsPathTraversal 检查字符串是否包含路径遍历序列
func ContainsPathTraversal(s string) bool {
	s = strings.ToLower(s)
	dangerousPatterns := []string{
		"../", "..\\",
		"/..", "\\..",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
