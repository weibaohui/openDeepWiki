package model

import "time"

// AgentVersion Agent 版本记录
// 用于追踪 backend/agents/ 目录下 YAML 文件的变更历史
type AgentVersion struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	FileName          string     `json:"file_name" gorm:"size:255;index:idx_file_name,priority:1;not null"`          // Agent 文件名（如 markdown_checker.yaml）
	Content           string     `json:"content" gorm:"type:text;not null"`                                     // YAML 文件内容
	Version           int        `json:"version" gorm:"not null;index:idx_file_version,priority:1"`             // 版本号（每个文件独立计数）
	SavedAt           time.Time  `json:"saved_at" gorm:"not null"`                                                  // 保存时间
	Source            string     `json:"source" gorm:"size:50;not null;default:'web'"`                          // 来源：web/file_change
	RestoreFromVersion *int       `json:"restore_from_version"`                                                      // 如果是恢复操作，记录源版本号
	CreatedAt         time.Time  `json:"created_at" gorm:"not null"`
}

// TableName 指定表名
func (AgentVersion) TableName() string {
	return "agent_versions"
}

// TableNameWithGormTableName 指定 GORM 表名
func (AgentVersion) GormTableName() string {
	return "agent_versions"
}
