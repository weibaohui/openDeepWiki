package model

import (
	"time"
	"gorm.io/gorm"
)

// APIKey API Key 配置
type APIKey struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	Name             string     `json:"name" gorm:"size:255;uniqueIndex;not null"`
	Provider         string     `json:"provider" gorm:"size:50;index:idx_api_keys_provider;not null"`
	BaseURL          string     `json:"base_url" gorm:"size:500;not null"`
	APIKey           string     `json:"api_key" gorm:"type:text;not null"`
	Model            string     `json:"model" gorm:"size:255;not null"`
	Priority         int        `json:"priority" gorm:"default:0;index:idx_api_keys_priority"`
	Status           string     `json:"status" gorm:"size:20;default:'enabled';index:idx_api_keys_status"` // enabled/disabled/unavailable
	RequestCount     int        `json:"request_count" gorm:"default:0"`
	ErrorCount       int        `json:"error_count" gorm:"default:0"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"index:idx_api_keys_deleted_at"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}

// MaskAPIKey 脱敏 API Key（只显示前3位和后4位）
func (a *APIKey) MaskAPIKey() string {
	if len(a.APIKey) <= 7 {
		return "***"
	}
	return a.APIKey[:3] + "***" + a.APIKey[len(a.APIKey)-4:]
}

// IsAvailable 检查是否可用
func (a *APIKey) IsAvailable() bool {
	if a.Status != "enabled" {
		return false
	}
	if a.RateLimitResetAt != nil && a.RateLimitResetAt.After(time.Now()) {
		return false
	}
	return true
}

// BeforeUpdate GORM 钩子：更新前自动设置 UpdatedAt
func (a *APIKey) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now()
	return nil
}
