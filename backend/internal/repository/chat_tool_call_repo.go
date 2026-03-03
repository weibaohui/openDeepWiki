package repository

import (
	"context"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// ChatToolCallRepository 工具调用仓储接口
type ChatToolCallRepository interface {
	Create(ctx context.Context, toolCall *model.ChatToolCall) error
	CreateOrUpdate(ctx context.Context, toolCall *model.ChatToolCall) error
	GetByToolCallID(ctx context.Context, toolCallID string) (*model.ChatToolCall, error)
	ListByMessageID(ctx context.Context, messageID string) ([]*model.ChatToolCall, error)
	UpdateResult(ctx context.Context, toolCallID, result string, durationMs int) error
	UpdateStatus(ctx context.Context, toolCallID, status string) error
}

// chatToolCallRepository 实现
type chatToolCallRepository struct {
	db *gorm.DB
}

// NewChatToolCallRepository 创建仓储实例
func NewChatToolCallRepository(db *gorm.DB) ChatToolCallRepository {
	return &chatToolCallRepository{db: db}
}

// Create 创建工具调用记录
func (r *chatToolCallRepository) Create(ctx context.Context, toolCall *model.ChatToolCall) error {
	return r.db.WithContext(ctx).Create(toolCall).Error
}

// CreateOrUpdate 创建或更新工具调用记录（Upsert）
func (r *chatToolCallRepository) CreateOrUpdate(ctx context.Context, toolCall *model.ChatToolCall) error {
	return r.db.WithContext(ctx).
		Where("tool_call_id = ?", toolCall.ToolCallID).
		Assign(toolCall).
		FirstOrCreate(toolCall).Error
}

// GetByToolCallID 根据toolCallID获取工具调用
func (r *chatToolCallRepository) GetByToolCallID(ctx context.Context, toolCallID string) (*model.ChatToolCall, error) {
	var toolCall model.ChatToolCall
	err := r.db.WithContext(ctx).
		Where("tool_call_id = ?", toolCallID).
		First(&toolCall).Error
	if err != nil {
		return nil, err
	}
	return &toolCall, nil
}

// ListByMessageID 获取消息的工具调用列表
func (r *chatToolCallRepository) ListByMessageID(ctx context.Context, messageID string) ([]*model.ChatToolCall, error) {
	var toolCalls []*model.ChatToolCall
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Order("created_at ASC").
		Find(&toolCalls).Error
	if err != nil {
		return nil, err
	}
	return toolCalls, nil
}

// UpdateResult 更新工具调用结果
func (r *chatToolCallRepository) UpdateResult(ctx context.Context, toolCallID, result string, durationMs int) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.ChatToolCall{}).
		Where("tool_call_id = ?", toolCallID).
		Updates(map[string]interface{}{
			"result":       result,
			"status":       "completed",
			"completed_at": now,
			"duration_ms":  durationMs,
		}).Error
}

// UpdateStatus 更新工具调用状态
func (r *chatToolCallRepository) UpdateStatus(ctx context.Context, toolCallID, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatToolCall{}).
		Where("tool_call_id = ?", toolCallID).
		Update("status", status).Error
}
