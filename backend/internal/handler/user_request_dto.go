package handler

import "time"

// UserRequestDTO 用户需求请求DTO
type UserRequestDTO struct {
	Content string `json:"content" binding:"required,max=200"`
}

// UserRequestResponseDTO 用户需求响应DTO
type UserRequestResponseDTO struct {
	ID           uint      `json:"id"`
	RepositoryID uint      `json:"repository_id"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
