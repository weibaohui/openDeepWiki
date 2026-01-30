package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/opendeepwiki/backend/internal/pkg/llm"
)

// SkillConfig Skill 配置文件结构
type SkillConfig struct {
	// 基础信息
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`

	// Provider 配置
	Provider string `yaml:"provider" json:"provider"` // builtin / http

	// HTTP Provider 特有配置
	Endpoint string            `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Timeout  int               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// 安全相关
	RiskLevel string `yaml:"risk_level,omitempty" json:"risk_level,omitempty"` // read / write / destructive

	// 参数定义
	Parameters llm.ParameterSchema `yaml:"parameters" json:"parameters"`
}

// Validate 校验配置是否有效
func (c *SkillConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: skill name is required", ErrInvalidConfig)
	}

	if c.Description == "" {
		return fmt.Errorf("%w: skill description is required", ErrInvalidConfig)
	}

	if c.Provider == "" {
		return fmt.Errorf("%w: skill provider is required", ErrInvalidConfig)
	}

	if c.Provider == "http" && c.Endpoint == "" {
		return fmt.Errorf("%w: endpoint is required for http provider", ErrInvalidConfig)
	}

	// 校验 RiskLevel
	if c.RiskLevel != "" && c.RiskLevel != "read" && c.RiskLevel != "write" && c.RiskLevel != "destructive" {
		return fmt.Errorf("%w: invalid risk_level: %s", ErrInvalidConfig, c.RiskLevel)
	}

	return nil
}

// ResolveSkillsDir 解析 Skills 目录
// 优先级：环境变量 > 配置文件 > 默认目录
func ResolveSkillsDir(configDir string) (string, error) {
	// 1. 检查环境变量
	if dir := os.Getenv("SKILLS_DIR"); dir != "" {
		return filepath.Abs(dir)
	}

	// 2. 检查配置文件
	if configDir != "" {
		return filepath.Abs(configDir)
	}

	// 3. 默认目录（与可执行文件同级）
	exePath, err := os.Executable()
	if err != nil {
		// 如果无法获取可执行文件路径，使用当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, "skills"), nil
	}

	defaultDir := filepath.Join(filepath.Dir(exePath), "skills")
	return defaultDir, nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}
