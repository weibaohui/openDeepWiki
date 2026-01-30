package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigLoader 配置加载器
type ConfigLoader struct {
	registry  Registry
	providers *ProviderRegistry
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(registry Registry, providers *ProviderRegistry) *ConfigLoader {
	return &ConfigLoader{
		registry:  registry,
		providers: providers,
	}
}

// LoadFromDir 从目录加载所有配置
func (l *ConfigLoader) LoadFromDir(dir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 目录不存在，静默返回
		return nil
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return err
	}

	ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return err
	}
	files = append(files, ymlFiles...)

	jsonFiles, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	files = append(files, jsonFiles...)

	for _, file := range files {
		if err := l.LoadFromFile(file); err != nil {
			// 记录错误但继续加载其他文件
			fmt.Fprintf(os.Stderr, "Failed to load skill config from %s: %v\n", file, err)
			continue
		}
	}

	return nil
}

// LoadFromFile 从文件加载配置
func (l *ConfigLoader) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var config SkillConfig

	// 根据扩展名解析
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = json.Unmarshal(data, &config)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// 校验配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 获取 Provider
	provider, err := l.providers.Get(config.Provider)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// 创建 Skill
	skill, err := provider.Create(config)
	if err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	// 注册到 Registry
	// 如果已存在，先注销（支持更新）
	if _, err := l.registry.Get(config.Name); err == nil {
		_ = l.registry.Unregister(config.Name)
	}

	if err := l.registry.Register(skill); err != nil {
		return fmt.Errorf("failed to register skill: %w", err)
	}

	return nil
}

// UnloadFromFile 根据文件路径卸载 Skill
func (l *ConfigLoader) UnloadFromFile(path string) error {
	// 从文件名推断 Skill 名称
	basename := filepath.Base(path)
	ext := filepath.Ext(basename)
	name := basename[:len(basename)-len(ext)]

	// 首先尝试使用文件名（不含扩展名）作为 Skill 名称卸载
	// 这在大部分情况下是正确的
	err := l.registry.Unregister(name)
	if err == nil {
		return nil
	}

	// 如果失败，尝试从文件内容读取 name 字段
	data, err := os.ReadFile(path)
	if err != nil {
		// 文件可能已被删除，无法读取内容
		// 尝试从 Registry 中查找匹配的 skill
		// 由于无法确定确切的 name，这里返回错误
		return fmt.Errorf("cannot determine skill name from deleted file: %s", path)
	}

	// 尝试解析获取 name
	var config struct {
		Name string `yaml:"name" json:"name"`
	}

	ext = strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		_ = yaml.Unmarshal(data, &config)
	case ".json":
		_ = json.Unmarshal(data, &config)
	}

	if config.Name != "" {
		return l.registry.Unregister(config.Name)
	}

	// 最后尝试使用文件名
	return l.registry.Unregister(name)
}
