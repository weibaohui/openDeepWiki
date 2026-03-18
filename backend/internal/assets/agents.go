// Package assets 提供内嵌的 agents 资源文件
// 在编译时将 backend/agents 目录下的所有 YAML 文件打包进二进制
// 运行时可自动释放到指定目录
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"k8s.io/klog/v2"
)

//go:embed all:agents/*.yaml
var agentsFS embed.FS

// ExtractAgents 将内嵌的 agents 文件释放到指定目录。
//
// 参数:
//   - targetDir: 目标目录路径
//
// 返回:
//   - 错误信息（如果有）
//
// 说明:
//   - 如果目标目录已存在且有文件，不会覆盖现有文件
//   - 仅释放不存在的文件，保护用户自定义配置
func ExtractAgents(targetDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory %s: %w", targetDir, err)
	}

	// 读取内嵌文件系统
	err := fs.WalkDir(agentsFS, "agents", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		// 只处理 .yaml 文件
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		// 读取内嵌文件内容
		content, err := agentsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// 目标文件路径
		fileName := filepath.Base(path)
		targetPath := filepath.Join(targetDir, fileName)

		// 如果文件已存在，跳过（不覆盖用户修改）
		if _, err := os.Stat(targetPath); err == nil {
			klog.V(6).Infof("[Assets] Agent file already exists, skipping: %s", fileName)
			return nil
		}

		// 写入文件
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write agent file %s: %w", targetPath, err)
		}

		klog.V(6).Infof("[Assets] Extracted agent: %s -> %s", fileName, targetPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract agents: %w", err)
	}

	klog.V(6).Infof("[Assets] Agents extracted to: %s", targetDir)
	return nil
}

// ListEmbeddedAgents 列出所有内嵌的 agents 文件名。
//
// 返回:
//   - 文件名列表
//   - 错误信息（如果有）
//
// 说明:
//   - 仅返回 .yaml 后缀的文件
//   - 返回的是文件名（不含路径）
func ListEmbeddedAgents() ([]string, error) {
	var files []string

	err := fs.WalkDir(agentsFS, "agents", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, filepath.Base(path))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
