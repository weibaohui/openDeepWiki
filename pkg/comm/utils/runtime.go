package utils

import (
	"os"
	"path/filepath"
)

// EnsureRuntimeDir 确保运行时目录存在
func EnsureRuntimeDir(repoName string) (string, error) {
	// 获取项目根目录下的 data/runtime 目录
	runtimeDir := filepath.Join("data", "runtime", repoName)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return "", err
	}

	return runtimeDir, nil
}

// GetRuntimeFilePath 获取运行时文件的完整路径
func GetRuntimeFilePath(repoName, filename string) (string, error) {
	runtimeDir, err := EnsureRuntimeDir(repoName)
	if err != nil {
		return "", err
	}

	return filepath.Join(runtimeDir, filename), nil
}
