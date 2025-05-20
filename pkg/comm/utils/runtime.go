package utils

import (
	"os"
	"path/filepath"
)

// EnsureRuntimeDir 确保为指定仓库名创建并返回运行时目录的完整路径。
// 如果目录不存在，则以 0755 权限递归创建；若创建失败，返回错误。
func EnsureRuntimeDir(repoName string) (string, error) {
	// 获取项目根目录下的 data/runtime 目录
	runtimeDir := filepath.Join("data", "runtime", repoName)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return "", err
	}

	return runtimeDir, nil
}

// GetRuntimeFilePath 返回指定仓库运行时目录下指定文件的完整路径。
// 首先确保对应的运行时目录已创建，若目录创建失败则返回错误，否则返回文件的完整路径。
func GetRuntimeFilePath(repoName, filename string) (string, error) {
	runtimeDir, err := EnsureRuntimeDir(repoName)
	if err != nil {
		return "", err
	}

	return filepath.Join(runtimeDir, filename), nil
}
