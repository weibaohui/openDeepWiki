package syncdto

import "time"

type StartRequest struct {
	TargetServer string `json:"target_server" binding:"required"`
	RepositoryID uint   `json:"repository_id" binding:"required"`
}

type StartResponse struct {
	Code string        `json:"code"`
	Data StartData     `json:"data"`
	Meta *ResponseMeta `json:"meta,omitempty"`
}

type StartData struct {
	SyncID       string `json:"sync_id"`
	RepositoryID uint   `json:"repository_id"`
	TotalTasks   int    `json:"total_tasks"`
	Status       string `json:"status"`
}

type StatusResponse struct {
	Code string        `json:"code"`
	Data StatusData    `json:"data"`
	Meta *ResponseMeta `json:"meta,omitempty"`
}

type StatusData struct {
	SyncID         string    `json:"sync_id"`
	RepositoryID   uint      `json:"repository_id"`
	TotalTasks     int       `json:"total_tasks"`
	CompletedTasks int       `json:"completed_tasks"`
	FailedTasks    int       `json:"failed_tasks"`
	Status         string    `json:"status"`
	CurrentTask    string    `json:"current_task"`
	StartedAt      time.Time `json:"started_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PingResponse struct {
	Code string `json:"code"`
}

type RepositoryUpsertRequest struct {
	RepositoryID uint      `json:"repository_id" binding:"required"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Description  string    `json:"description"`
	CloneBranch  string    `json:"clone_branch"`
	CloneCommit  string    `json:"clone_commit_id"`
	SizeMB       float64   `json:"size_mb"`
	Status       string    `json:"status"`
	ErrorMsg     string    `json:"error_msg"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RepositoryUpsertResponse struct {
	Code string               `json:"code"`
	Data RepositoryUpsertData `json:"data"`
	Meta *ResponseMeta        `json:"meta,omitempty"`
}

type RepositoryUpsertData struct {
	RepositoryID uint   `json:"repository_id"`
	Name         string `json:"name"`
}

type TaskCreateRequest struct {
	RepositoryID uint       `json:"repository_id" binding:"required"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	ErrorMsg     string     `json:"error_msg"`
	SortOrder    int        `json:"sort_order"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type TaskCreateResponse struct {
	Code string         `json:"code"`
	Data TaskCreateData `json:"data"`
	Meta *ResponseMeta  `json:"meta,omitempty"`
}

type TaskCreateData struct {
	TaskID       uint   `json:"task_id"`
	RepositoryID uint   `json:"repository_id"`
	Title        string `json:"title"`
}

type DocumentCreateRequest struct {
	RepositoryID uint      `json:"repository_id" binding:"required"`
	TaskID       uint      `json:"task_id" binding:"required"`
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	Content      string    `json:"content"`
	SortOrder    int       `json:"sort_order"`
	Version      int       `json:"version"`
	IsLatest     bool      `json:"is_latest"`
	ReplacedBy   uint      `json:"replaced_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DocumentCreateResponse struct {
	Code string             `json:"code"`
	Data DocumentCreateData `json:"data"`
	Meta *ResponseMeta      `json:"meta,omitempty"`
}

type DocumentCreateData struct {
	DocumentID   uint `json:"document_id"`
	RepositoryID uint `json:"repository_id"`
	TaskID       uint `json:"task_id"`
}

type TaskUpdateDocIDRequest struct {
	TaskID     uint `json:"task_id" binding:"required"`
	DocumentID uint `json:"document_id" binding:"required"`
}

type TaskUpdateDocIDResponse struct {
	Code string              `json:"code"`
	Data TaskUpdateDocIDData `json:"data"`
	Meta *ResponseMeta       `json:"meta,omitempty"`
}

type TaskUpdateDocIDData struct {
	TaskID     uint `json:"task_id"`
	DocumentID uint `json:"document_id"`
}

type ResponseMeta struct {
	Message string `json:"message,omitempty"`
}
