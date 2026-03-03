package model

import (
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
)

type Repository struct {
	ID                    uint       `json:"id" gorm:"primaryKey"`
	Name                  string     `json:"name" gorm:"size:255;"`
	URL                   string     `json:"url" gorm:"size:500;"`
	LocalPath             string     `json:"local_path" gorm:"size:500"`
	Description           string     `json:"description" gorm:"size:1000"`
	CloneBranch           string     `json:"clone_branch" gorm:"size:255"`
	CloneCommit           string     `json:"clone_commit_id" gorm:"size:100"`
	SizeMB                float64    `json:"size_mb" gorm:"default:0"`
	Status                string     `json:"status" gorm:"size:50;default:pending"` // pending, cloning, ready, analyzing, completed, error
	ErrorMsg              string     `json:"error_msg" gorm:"size:1000"`
	NextUpdateTime        *time.Time `json:"next_update_time,omitempty"`     // 下次更新时间（基于活跃度动态调整）
	TodayActivityCount    int        `json:"today_activity_count,omitempty"`   // 今日活跃度点数
	LastActivityResetDate *time.Time `json:"last_activity_reset_date,omitempty"` // 上次重置活跃度的日期
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	Tasks                 []Task     `json:"tasks,omitempty" gorm:"foreignKey:RepositoryID"`
	Documents             []Document `json:"documents,omitempty" gorm:"foreignKey:RepositoryID"`
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
	ID            uint      `json:"id" gorm:"primaryKey"`
	RepositoryID  uint      `json:"repository_id" gorm:"index;"`
	TaskID        uint      `json:"task_id" gorm:"index"`
	Title         string    `json:"title" gorm:"size:255;"`
	Filename      string    `json:"filename" gorm:"size:255;"`
	Content       string    `json:"content" gorm:"type:text"`
	SortOrder     int       `json:"sort_order" gorm:"default:0"`
	Version       int       `json:"version" gorm:"default:1;index"`
	IsLatest      bool      `json:"is_latest" gorm:"index"`
	ReplacedBy    uint      `json:"replaced_by" gorm:"index;"` //被替换为哪个DocID
	CloneBranch   string    `json:"clone_branch" gorm:"size:255"`   // 生成文档时的分支名称
	CloneCommitID string    `json:"clone_commit_id" gorm:"size:100"` // 生成文档时的 commit id
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

type SyncTarget struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	URL       string    `json:"url" gorm:"size:500;uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SyncEvent struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	EventType    string    `json:"event_type" gorm:"size:50;index;not null"`
	RepositoryID uint      `json:"repository_id" gorm:"index;not null"`
	DocID        uint      `json:"doc_id" gorm:"index;not null"`
	TargetServer string    `json:"target_server" gorm:"size:500"`
	Success      bool      `json:"success" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
}

type IncrementalUpdateHistory struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RepositoryID uint      `json:"repository_id" gorm:"index;not null"`
	BaseCommit   string    `json:"base_commit" gorm:"size:100;not null"`
	LatestCommit string    `json:"latest_commit" gorm:"size:100;not null"`
	AddedDirs    int       `json:"added_dirs" gorm:"default:0"`
	UpdatedDirs  int       `json:"updated_dirs" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
}

// DocumentVector 文档向量模型，用于存储文档的向量嵌入
type DocumentVector struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	DocumentID  uint      `json:"document_id" gorm:"index:idx_doc_vectors_document_id;not null"`
	ModelName   string    `json:"model_name" gorm:"size:100;index:idx_doc_vectors_model;not null"`
	Vector      []float32 `json:"-" gorm:"type:blob;not null"` // BLOB 存储向量数据，不直接 JSON 序列化
	Dimension   int       `json:"dimension" gorm:"not null"`
	GeneratedAt time.Time `json:"generated_at" gorm:"not null"`
	Metadata    string    `json:"metadata" gorm:"type:text"` // JSON 格式的额外元数据
	Document    *Document `json:"document,omitempty" gorm:"foreignKey:DocumentID"`
}

// VectorTask 向量生成任务模型，用于跟踪向量生成的状态
type VectorTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	DocumentID   uint       `json:"document_id" gorm:"index;not null"`
	Status       string     `json:"status" gorm:"size:20;index;not null;default:'pending'"` // pending, processing, completed, failed
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Document     *Document  `json:"document,omitempty" gorm:"foreignKey:DocumentID"`
}

// ChatSession 对话会话表
type ChatSession struct {
	ID        uint          `json:"id" gorm:"primaryKey"`
	SessionID string        `json:"session_id" gorm:"size:64;uniqueIndex"`  // 唯一会话标识
	RepoID    uint          `json:"repo_id" gorm:"index"`                   // 关联仓库ID
	Title     string        `json:"title" gorm:"size:255"`                  // 会话标题
	Status    string        `json:"status" gorm:"size:20;default:'active'"` // active, archived, deleted
	Messages  []ChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;references:SessionID"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ChatMessage 对话消息表
type ChatMessage struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	SessionID   string         `json:"session_id" gorm:"size:64;index"`                // 所属会话
	MessageID   string         `json:"message_id" gorm:"size:64;uniqueIndex"`          // 唯一消息标识
	ParentID    *string        `json:"parent_id" gorm:"size:64;index"`                 // 父消息ID（用于支持分支对话）
	Role        string         `json:"role" gorm:"size:20"`                            // user, assistant, system, tool
	Content     string         `json:"content" gorm:"type:text"`                       // 消息内容
	ContentType string         `json:"content_type" gorm:"size:20;default:'text'"`     // text, thinking, tool_call, tool_result
	ToolCalls   []ChatToolCall `json:"tool_calls,omitempty" gorm:"foreignKey:MessageID"` // 关联的工具调用
	Model       string         `json:"model" gorm:"size:50"`                           // 使用的模型
	TokenUsed   int            `json:"token_used" gorm:"default:0"`                    // Token使用量
	Status      string         `json:"status" gorm:"size:20;default:'completed'"`      // pending, streaming, completed, stopped, error
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at"`
}

// ChatToolCall 工具调用表
type ChatToolCall struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	MessageID   string     `json:"message_id" gorm:"size:64;index"`         // 所属消息
	ToolCallID  string     `json:"tool_call_id" gorm:"size:64;uniqueIndex"` // 工具调用标识
	ToolName    string     `json:"tool_name" gorm:"size:100"`               // 工具名称
	Arguments   string     `json:"arguments" gorm:"type:text"`              // 调用参数（JSON）
	Result      string     `json:"result" gorm:"type:text"`                 // 执行结果
	Status      string     `json:"status" gorm:"size:20;default:'pending'"` // pending, running, completed, error
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	DurationMs  int        `json:"duration_ms" gorm:"default:0"`            // 执行耗时（毫秒）
}
