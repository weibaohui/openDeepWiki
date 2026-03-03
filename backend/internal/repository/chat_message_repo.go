package repository

import (
	"context"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// ChatMessageRepository 对话消息仓储接口
type ChatMessageRepository interface {
	Create(ctx context.Context, message *model.ChatMessage) error
	GetByMessageID(ctx context.Context, messageID string) (*model.ChatMessage, error)
	ListBySessionID(ctx context.Context, sessionID string, limit int, beforeID *string) ([]*model.ChatMessage, error)
	UpdateContent(ctx context.Context, messageID, content string) error
	Finalize(ctx context.Context, messageID string, tokenUsed int, status string, completedAt time.Time) error
	UpdateStatus(ctx context.Context, messageID, status string) error
	CountBySessionID(ctx context.Context, sessionID string) (int64, error)
}

// chatMessageRepository 实现
type chatMessageRepository struct {
	db *gorm.DB
}

// NewChatMessageRepository 创建仓储实例
func NewChatMessageRepository(db *gorm.DB) ChatMessageRepository {
	return &chatMessageRepository{db: db}
}

// Create 创建消息
func (r *chatMessageRepository) Create(ctx context.Context, message *model.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetByMessageID 根据messageID获取消息
func (r *chatMessageRepository) GetByMessageID(ctx context.Context, messageID string) (*model.ChatMessage, error) {
	var message model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// ListBySessionID 获取会话的消息列表
func (r *chatMessageRepository) ListBySessionID(ctx context.Context, sessionID string, limit int, beforeID *string) ([]*model.ChatMessage, error) {
	var messages []*model.ChatMessage

	query := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Preload("ToolCalls").
		Order("created_at DESC")

	// 如果指定了beforeID，只查询在该消息之前的消息
	if beforeID != nil && *beforeID != "" {
		var beforeMessage model.ChatMessage
		if err := r.db.WithContext(ctx).
			Where("message_id = ?", *beforeID).
			First(&beforeMessage).Error; err == nil {
			query = query.Where("created_at < ?", beforeMessage.CreatedAt)
		}
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// 反转顺序，按时间正序返回
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// UpdateContent 更新消息内容（流式更新）
func (r *chatMessageRepository) UpdateContent(ctx context.Context, messageID, content string) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("message_id = ?", messageID).
		Update("content", content).Error
}

// Finalize 完成消息保存
func (r *chatMessageRepository) Finalize(ctx context.Context, messageID string, tokenUsed int, status string, completedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("message_id = ?", messageID).
		Updates(map[string]interface{}{
			"token_used":   tokenUsed,
			"status":       status,
			"completed_at": completedAt,
		}).Error
}

// UpdateStatus 更新消息状态
func (r *chatMessageRepository) UpdateStatus(ctx context.Context, messageID, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("message_id = ?", messageID).
		Update("status", status).Error
}

// CountBySessionID 统计会话消息数量
func (r *chatMessageRepository) CountBySessionID(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error
	return count, err
}
