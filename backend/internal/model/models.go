package model

import (
	"time"
)

type Repository struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"size:255;"`
	URL         string     `json:"url" gorm:"size:500;"`
	LocalPath   string     `json:"local_path" gorm:"size:500"`
	Description string     `json:"description" gorm:"size:1000"`
	CloneBranch string     `json:"clone_branch" gorm:"size:255"`
	CloneCommit string     `json:"clone_commit_id" gorm:"size:100"`
	SizeMB      float64    `json:"size_mb" gorm:"default:0"`
	Status      string     `json:"status" gorm:"size:50;default:pending"` // pending, cloning, ready, analyzing, completed, error
	ErrorMsg    string     `json:"error_msg" gorm:"size:1000"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Tasks       []Task     `json:"tasks,omitempty" gorm:"foreignKey:RepositoryID"`
	Documents   []Document `json:"documents,omitempty" gorm:"foreignKey:RepositoryID"`
}

type Task struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	RepositoryID uint       `json:"repository_id" gorm:"index;"`
	Type         string     `json:"type" gorm:"size:50;"` // overview, architecture, api, business-flow, deployment
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
	RepositoryID uint      `json:"repository_id" gorm:"index;"`
	TaskID       uint      `json:"task_id" gorm:"index"`
	Title        string    `json:"title" gorm:"size:255;"`
	Filename     string    `json:"filename" gorm:"size:255;"`
	Content      string    `json:"content" gorm:"type:text"`
	SortOrder    int       `json:"sort_order" gorm:"default:0"`
	Version      int       `json:"version" gorm:"default:1;index"`
	IsLatest     bool      `json:"is_latest" gorm:"default:true;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TaskEvidence struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RepositoryID uint      `json:"repository_id" gorm:"index;"`
	TaskID       uint      `json:"task_id" gorm:"index;"`
	Title        string    `json:"title" gorm:"size:255;"`
	Aspect       string    `json:"aspect" gorm:"size:255;"`
	Source       string    `json:"source" gorm:"size:500;"`
	Detail       string    `json:"detail" gorm:"type:text"`
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
}
