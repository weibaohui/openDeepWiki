package skills

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config Manager 配置
type Config struct {
	Dir            string
	AutoReload     bool
	ReloadInterval time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Dir:            "./skills",
		AutoReload:     true,
		ReloadInterval: 5 * time.Second,
	}
}

// Manager Skills 管理器
type Manager struct {
	Config   *Config
	Registry Registry
	Parser   *Parser
	Loader   *Loader
	Matcher  *Matcher
	Injector *Injector
	watcher  *FileWatcher
}

// NewManager 创建 Manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 解析目录
	dir, err := resolveSkillsDir(config.Dir)
	if err != nil {
		return nil, err
	}
	config.Dir = dir

	log.Printf("Skills directory: %s", dir)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	// 创建组件
	registry := NewRegistry()
	parser := NewParser()
	loader := NewLoader(parser, registry)
	matcher := NewMatcher(registry)
	injector := NewInjector(loader)

	m := &Manager{
		Config:   config,
		Registry: registry,
		Parser:   parser,
		Loader:   loader,
		Matcher:  matcher,
		Injector: injector,
	}

	// 初始加载
	results, err := loader.LoadFromDir(dir)
	if err != nil {
		log.Printf("Warning: failed to load skills: %v", err)
	} else {
		loaded := 0
		updated := 0
		failed := 0
		for _, r := range results {
			switch r.Action {
			case "created":
				loaded++
			case "updated":
				updated++
			case "failed":
				failed++
			}
		}
		if loaded > 0 || updated > 0 {
			log.Printf("Loaded %d skills, updated %d skills", loaded, updated)
		}
		if failed > 0 {
			log.Printf("Failed to load %d skills", failed)
		}
	}

	// 启动热加载
	if config.AutoReload {
		m.startWatcher()
	}

	return m, nil
}

// startWatcher 启动文件监听
func (m *Manager) startWatcher() {
	m.watcher = NewFileWatcher(m.Config.Dir, m.Config.ReloadInterval, func(event FileEvent) {
		switch event.Type {
		case "create":
			log.Printf("Loading new skill from %s", event.Path)
			if _, err := m.Loader.LoadFromPath(event.Path); err != nil {
				log.Printf("Failed to load skill: %v", err)
			} else {
				log.Printf("Successfully loaded skill from %s", event.Path)
			}

		case "modify":
			skillName := filepath.Base(event.Path)
			log.Printf("Reloading skill: %s", skillName)
			if _, err := m.Loader.Reload(skillName); err != nil {
				log.Printf("Failed to reload skill: %v", err)
			} else {
				log.Printf("Successfully reloaded skill: %s", skillName)
			}

		case "delete":
			skillName := filepath.Base(event.Path)
			log.Printf("Unloading skill: %s", skillName)
			if err := m.Loader.Unload(skillName); err != nil {
				log.Printf("Failed to unload skill: %v", err)
			} else {
				log.Printf("Successfully unloaded skill: %s", skillName)
			}
		}
	})

	if err := m.watcher.Start(); err != nil {
		log.Printf("Warning: failed to start file watcher: %v", err)
	}
}

// Stop 停止 Manager
func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
}

// MatchAndInject 匹配 Skills 并注入 Prompt
func (m *Manager) MatchAndInject(systemPrompt string, task Task) (string, []*Match, error) {
	// 匹配 Skills
	matches, err := m.Matcher.Match(task)
	if err != nil {
		return "", nil, err
	}

	// 如果没有匹配到，返回原始 prompt
	if len(matches) == 0 {
		return systemPrompt, nil, nil
	}

	// 注入 Prompt
	newPrompt, err := m.Injector.InjectToPrompt(systemPrompt, matches)
	if err != nil {
		return "", nil, err
	}

	return newPrompt, matches, nil
}

// MatchAndInjectByDescription 根据描述匹配并注入
func (m *Manager) MatchAndInjectByDescription(systemPrompt string, description string) (string, []*Match, error) {
	task := Task{Description: description}
	return m.MatchAndInject(systemPrompt, task)
}

// ReloadAll 重新加载所有 Skills
func (m *Manager) ReloadAll() error {
	// 获取当前所有 skills
	skills := m.Registry.List()

	// 卸载所有
	for _, skill := range skills {
		if err := m.Loader.Unload(skill.Name); err != nil {
			log.Printf("Failed to unload skill %s: %v", skill.Name, err)
		}
	}

	// 重新加载
	_, err := m.Loader.LoadFromDir(m.Config.Dir)
	return err
}

// GetSkillContent 获取 Skill 完整内容
func (m *Manager) GetSkillContent(name string) (*Skill, string, error) {
	skill, err := m.Registry.Get(name)
	if err != nil {
		return nil, "", err
	}

	body, err := m.Loader.GetBody(name)
	if err != nil {
		return nil, "", err
	}

	return skill, body, nil
}

// resolveSkillsDir 解析 Skills 目录
func resolveSkillsDir(configDir string) (string, error) {
	// 1. 环境变量
	if dir := os.Getenv("SKILLS_DIR"); dir != "" {
		return filepath.Abs(dir)
	}

	// 2. 配置
	if configDir != "" {
		return filepath.Abs(configDir)
	}

	// 3. 默认
	exePath, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "skills"), nil
	}
	return filepath.Join(filepath.Dir(exePath), "skills"), nil
}
