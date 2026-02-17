package model

import "time"

// UserRequest 用户需求模型
// 用于存储用户提交的内容需求，记录用户想要查看或了解的内容
type UserRequest struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RepositoryID uint      `json:"repository_id" gorm:"index;not null"`
	Content      string    `json:"content" gorm:"type:text;not null;size:200"`
	Status       string    `json:"status" gorm:"size:50;not null;default:pending"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	Repository *Repository `json:"repository,omitempty" gorm:"foreignKey:RepositoryID"`
}

// UserRequestStatus 用户需求状态枚举
const (
	UserRequestStatusPending    = "pending"    // 待处理
	UserRequestStatusProcessing = "processing" // 处理中
	UserRequestStatusCompleted  = "completed"  // 已完成
	UserRequestStatusRejected   = "rejected"   // 已拒绝
)
