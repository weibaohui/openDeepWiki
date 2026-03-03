package repository

import (
	"context"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// ChatSessionRepository 对话会话仓储接口
type ChatSessionRepository interface {
	Create(ctx context.Context, session *model.ChatSession) error
	GetBySessionID(ctx context.Context, sessionID string) (*model.ChatSession, error)
	ListByRepoID(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error)
	Update(ctx context.Context, session *model.ChatSession) error
	Delete(ctx context.Context, sessionID string) error
	UpdateTitle(ctx context.Context, sessionID, title string) error
}

// chatSessionRepository 实现
type chatSessionRepository struct {
	db *gorm.DB
}

// NewChatSessionRepository 创建仓储实例
func NewChatSessionRepository(db *gorm.DB) ChatSessionRepository {
	return &chatSessionRepository{db: db}
}

// Create 创建会话
func (r *chatSessionRepository) Create(ctx context.Context, session *model.ChatSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// GetBySessionID 根据sessionID获取会话
func (r *chatSessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListByRepoID 获取仓库的会话列表
func (r *chatSessionRepository) ListByRepoID(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error) {
	var sessions []*model.ChatSession
	var total int64

	// 查询总数
	if err := r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("repo_id = ? AND status != 'deleted'", repoID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Where("repo_id = ? AND status != 'deleted'", repoID).
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&sessions).Error
	if err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// Update 更新会话
func (r *chatSessionRepository) Update(ctx context.Context, session *model.ChatSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// Delete 删除会话（软删除）
func (r *chatSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("status", "deleted").Error
}

// UpdateTitle 更新会话标题
func (r *chatSessionRepository) UpdateTitle(ctx context.Context, sessionID, title string) error {
	return r.db.WithContext(ctx).
		Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		}).Error
}
