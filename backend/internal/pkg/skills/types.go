package skills

import (
	"time"
)

// Skill 技能定义
type Skill struct {
	// 元数据（始终加载）
	Name          string            `yaml:"name" json:"name"`
	Description   string            `yaml:"description" json:"description"`
	License       string            `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`

	// 路径信息
	Path          string `json:"path"`          // Skill 目录绝对路径
	SkillMDPath   string `json:"skill_md_path"` // SKILL.md 文件路径

	// 资源标志
	HasScripts    bool `json:"has_scripts"`
	HasReferences bool `json:"has_references"`
	HasAssets     bool `json:"has_assets"`

	// 状态
	Enabled  bool      `json:"enabled"`
	LoadedAt time.Time `json:"loaded_at"`
}

// Task 任务定义
type Task struct {
	Type        string   // 任务类型
	Description string   // 任务描述
	RepoType    string   // 仓库类型
	Tags        []string // 标签
}

// Match 匹配结果
type Match struct {
	Skill  *Skill
	Score  float64 // 匹配分数 0-1
	Reason string  // 匹配原因
}

// LoadResult 加载结果
type LoadResult struct {
	Skill  *Skill
	Error  error
	Action string // "created", "updated", "failed"
}
