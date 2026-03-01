package model

import (
	"time"

	"gorm.io/gorm"
)

// EmbeddingKey 嵌入模型配置
// 用于存储向量生成模型（如 OpenAI Embeddings）的配置信息
type EmbeddingKey struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	Name             string     `json:"name" gorm:"size:255;uniqueIndex;not null"`           // 配置名称，唯一
	Provider         string     `json:"provider" gorm:"size:50;index:idx_embedding_keys_provider;not null"` // 提供者类型：openai, ollama, http
	BaseURL          string     `json:"base_url" gorm:"size:500;not null"`              // API 地址
	APIKey           string     `json:"api_key" gorm:"type:text;not null"`                  // API Key
	Model            string     `json:"model" gorm:"size:255;not null"`                      // 模型名称
	Dimension        int        `json:"dimension" gorm:"not null;default:2560"`            // 向量维度，Qwen3-Embedding-4B 默认 2560 维
	Priority         int        `json:"priority" gorm:"default:0;index:idx_embedding_keys_priority"` // 优先级
	Status           string     `json:"status" gorm:"size:20;default:'enabled';index:idx_embedding_keys_status"` // enabled/disabled
	RequestCount     int        `json:"request_count" gorm:"default:0"`                     // 请求次数
	ErrorCount       int        `json:"error_count" gorm:"default:0"`                       // 错误次数
	LastUsedAt       *time.Time `json:"last_used_at"`                                     // 最后使用时间
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at"`                               // 限流重置时间
	Timeout          int        `json:"timeout" gorm:"default:30"`                          // 超时时间（秒）
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"index:idx_embedding_keys_deleted_at"`
}

// TableName 指定表名
func (EmbeddingKey) TableName() string {
	return "embedding_keys"
}

// MaskAPIKey 脱敏 API Key（只显示前3位和后4位）
func (e *EmbeddingKey) MaskAPIKey() string {
	if len(e.APIKey) <= 7 {
		return "***"
	}
	return e.APIKey[:3] + "***" + e.APIKey[len(e.APIKey)-4:]
}

// IsAvailable 检查是否可用
func (e *EmbeddingKey) IsAvailable() bool {
	if e.Status != "enabled" {
		return false
	}
	if e.RateLimitResetAt != nil && e.RateLimitResetAt.After(time.Now()) {
		return false
	}
	return true
}

// BeforeUpdate GORM 钩子：更新前自动设置 UpdatedAt
func (e *EmbeddingKey) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}