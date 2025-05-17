package models

import (
	"time"
)

type Repo struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"not null" json:"name"`  // 仓库名称
	Description string    `json:"description,omitempty"` // 仓库描述
	RepoType    string    `json:"repo_type"`             // 仓库类型（git/svn/local）
	URL         string    `json:"url"`                   // 仓库地址
	Branch      string    `json:"branch"`                // 默认分支
	CreatedBy   uint      `json:"created_by"`            // 创建人ID
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
