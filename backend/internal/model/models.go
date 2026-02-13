package model

import (
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
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
	ID           uint              `json:"id" gorm:"primaryKey"`
	RepositoryID uint              `json:"repository_id" gorm:"index;"`
	DocID        uint              `json:"doc_id" gorm:"index;"` // 关联的文档ID
	Repository   *Repository       `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
	WriterName   domain.WriterName `json:"writer_name" gorm:"size:255;default:DefaultWriter"` // 关联的写入器名称
	TaskType     domain.TaskType   `json:"task_type" gorm:"size:50;"`                         // 任务类型，生成文档，重写标题，生成目录
	Title        string            `json:"title" gorm:"type:text"`                            // 不限制，标题可以为空，可以重写
	Outline      string            `json:"outline" gorm:"type:text"`
	Status       string            `json:"status" gorm:"size:50;default:pending"` // pending, queued, running, succeeded, failed, canceled
	RunAfter     uint              `json:"run_after"`                             // 必须在哪个任务完成后才可以运行
	ErrorMsg     string            `json:"error_msg" gorm:"size:1000"`
	SortOrder    int               `json:"sort_order" gorm:"default:0"`
	StartedAt    *time.Time        `json:"started_at" gorm:"column:started_at"`
	CompletedAt  *time.Time        `json:"completed_at" gorm:"column:completed_at"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
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
	IsLatest     bool      `json:"is_latest" gorm:"index"`
	ReplacedBy   uint      `json:"replaced_by" gorm:"index;"` //被替换为哪个DocID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DocumentRating struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	DocumentID uint      `json:"document_id" gorm:"index"`
	Score      int       `json:"score"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type DocumentRatingStats struct {
	AverageScore float64 `json:"average_score"`
	RatingCount  int64   `json:"rating_count"`
}

type TaskHint struct {
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

type TaskUsage struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	TaskID           uint      `json:"task_id" gorm:"index;not null"`
	APIKeyName       string    `json:"api_key_name" gorm:"size:255;index;not null"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CachedTokens     int       `json:"cached_tokens"`
	ReasoningTokens  int       `json:"reasoning_tokens"`
	CreatedAt        time.Time `json:"created_at"`
}
