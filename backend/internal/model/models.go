package model

import (
	"time"
)

type Repository struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"size:255;not null"`
	URL         string     `json:"url" gorm:"size:500;not null"`
	LocalPath   string     `json:"local_path" gorm:"size:500"`
	Description string     `json:"description" gorm:"size:1000"`
	Status      string     `json:"status" gorm:"size:50;default:pending"` // pending, cloning, ready, analyzing, completed, error
	ErrorMsg    string     `json:"error_msg" gorm:"size:1000"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Tasks       []Task     `json:"tasks,omitempty" gorm:"foreignKey:RepositoryID"`
	Documents   []Document `json:"documents,omitempty" gorm:"foreignKey:RepositoryID"`
}

type Task struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	RepositoryID uint       `json:"repository_id" gorm:"index;not null"`
	Type         string     `json:"type" gorm:"size:50;not null"` // overview, architecture, api, business-flow, deployment
	Title        string     `json:"title" gorm:"size:255"`
	Status       string     `json:"status" gorm:"size:50;default:pending"` // pending, queued, running, succeeded, failed, canceled
	ErrorMsg     string     `json:"error_msg" gorm:"size:1000"`
	SortOrder    int        `json:"sort_order" gorm:"default:0"`
	StartedAt    *time.Time `json:"started_at" gorm:"column:started_at"`
	CompletedAt  *time.Time `json:"completed_at" gorm:"column:completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Document struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RepositoryID uint      `json:"repository_id" gorm:"index;not null"`
	TaskID       uint      `json:"task_id" gorm:"index"`
	Title        string    `json:"title" gorm:"size:255;not null"`
	Filename     string    `json:"filename" gorm:"size:255;not null"`
	Content      string    `json:"content" gorm:"type:text"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Task types definition
var TaskTypes = []struct {
	Type      string
	Title     string
	Filename  string
	SortOrder int
}{
	{"overview", "项目概览", "overview.md", 1},
	{"architecture", "架构分析", "architecture.md", 2},
	{"api", "核心接口", "api.md", 3},
	{"business-flow", "业务流程", "business-flow.md", 4},
	{"deployment", "部署配置", "deployment.md", 5},
}

// AIAnalysisTask AI分析任务模型
type AIAnalysisTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	RepositoryID uint       `json:"repository_id" gorm:"index;not null"`
	TaskID       string     `json:"task_id" gorm:"size:64;uniqueIndex"` // UUID
	Status       string     `json:"status" gorm:"size:50;default:pending"` // pending, running, completed, failed
	Progress     int        `json:"progress" gorm:"default:0"` // 0-100
	OutputPath   string     `json:"output_path" gorm:"size:500"`
	ErrorMsg     string     `json:"error_msg" gorm:"size:2000"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}
