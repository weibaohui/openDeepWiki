package skills

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader Skill 加载器
type Loader struct {
	parser      *Parser
	registry    Registry
	mu          sync.RWMutex
	skillBodies map[string]string // name -> body (缓存)
}

// NewLoader 创建加载器
func NewLoader(parser *Parser, registry Registry) *Loader {
	return &Loader{
		parser:      parser,
		registry:    registry,
		skillBodies: make(map[string]string),
	}
}

// LoadFromDir 从目录加载所有 Skills
func (l *Loader) LoadFromDir(dir string) ([]*LoadResult, error) {
	dir = filepath.Clean(dir)

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Skills directory does not exist: %s", dir)
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	results := make([]*LoadResult, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过以 . 开头的隐藏目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())

		// 检查是否是有效的 Skill 目录（包含 SKILL.md）
		skillMDPath := filepath.Join(skillPath, "SKILL.md")
		if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
			continue
		}

		result := l.loadSkill(skillPath)
		results = append(results, result)
	}

	return results, nil
}

// LoadFromPath 加载单个 Skill
func (l *Loader) LoadFromPath(path string) (*Skill, error) {
	result := l.loadSkill(path)
	if result.Error != nil {
		return nil, result.Error
	}
	return result.Skill, nil
}

// loadSkill 加载 Skill（内部）
func (l *Loader) loadSkill(path string) *LoadResult {
	skill, body, err := l.parser.Parse(path)
	if err != nil {
		return &LoadResult{
			Error:  err,
			Action: "failed",
		}
	}

	// 缓存 body
	l.mu.Lock()
	l.skillBodies[skill.Name] = body
	l.mu.Unlock()

	// 检查是否已存在
	existing, _ := l.registry.Get(skill.Name)
	action := "created"
	if existing != nil {
		action = "updated"
	}

	// 注册到 Registry
	if err := l.registry.Register(skill); err != nil {
		return &LoadResult{
			Skill:  skill,
			Error:  err,
			Action: "failed",
		}
	}

	return &LoadResult{
		Skill:  skill,
		Action: action,
	}
}

// GetBody 获取 Skill 的 body 内容
func (l *Loader) GetBody(name string) (string, error) {
	l.mu.RLock()
	body, exists := l.skillBodies[name]
	l.mu.RUnlock()

	if exists {
		return body, nil
	}

	// 如果缓存中没有，尝试从文件加载
	skill, err := l.registry.Get(name)
	if err != nil {
		return "", err
	}

	// 重新解析获取 body
	_, body, err = l.parser.Parse(skill.Path)
	if err != nil {
		return "", err
	}

	// 缓存
	l.mu.Lock()
	l.skillBodies[name] = body
	l.mu.Unlock()

	return body, nil
}

// LoadReferences 加载 references/ 下的文件
func (l *Loader) LoadReferences(skill *Skill) (map[string]string, error) {
	if !skill.HasReferences {
		return nil, nil
	}

	refsDir := filepath.Join(skill.Path, "references")
	entries, err := os.ReadDir(refsDir)
	if err != nil {
		return nil, err
	}

	refs := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(refsDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read reference file %s: %v", path, err)
			continue
		}
		refs[entry.Name()] = string(content)
	}

	return refs, nil
}

// LoadReference 加载单个 reference 文件
func (l *Loader) LoadReference(skill *Skill, filename string) (string, error) {
	if !skill.HasReferences {
		return "", fmt.Errorf("skill has no references")
	}

	path := filepath.Join(skill.Path, "references", filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// Unload 卸载 Skill
func (l *Loader) Unload(name string) error {
	l.mu.Lock()
	delete(l.skillBodies, name)
	l.mu.Unlock()

	return l.registry.Unregister(name)
}

// Reload 重新加载 Skill
func (l *Loader) Reload(name string) (*Skill, error) {
	skill, err := l.registry.Get(name)
	if err != nil {
		return nil, err
	}

	l.Unload(name)
	return l.LoadFromPath(skill.Path)
}
