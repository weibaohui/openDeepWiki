package service

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

// ChatService 对话服务接口
type ChatService interface {
	// 会话管理
	CreateSession(ctx context.Context, repoID uint) (*model.ChatSession, error)
	GetSession(ctx context.Context, sessionID string) (*model.ChatSession, error)
	ListSessions(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error)
	ListPublicSessions(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error)
	DeleteSession(ctx context.Context, sessionID string) error
	UpdateSessionTitle(ctx context.Context, sessionID, title string) error
	UpdateSessionVisibility(ctx context.Context, sessionID, visibility string) error
	UpdateMessageCount(ctx context.Context, sessionID string) error

	// 消息管理
	CreateUserMessage(ctx context.Context, sessionID, content string) (*model.ChatMessage, error)
	CreateAssistantMessage(ctx context.Context, sessionID string) (*model.ChatMessage, error)
	GetMessage(ctx context.Context, messageID string) (*model.ChatMessage, error)
	ListMessages(ctx context.Context, sessionID string, limit int, beforeID *string) ([]*model.ChatMessage, error)
	UpdateMessageContent(ctx context.Context, messageID, content string) error
	FinalizeMessage(ctx context.Context, messageID string, tokenUsed int, status string) error
	CountMessages(ctx context.Context, sessionID string) (int64, error)

	// 工具调用管理
	CreateToolCall(ctx context.Context, messageID, toolCallID, toolName, arguments string) (*model.ChatToolCall, error)
	CreateOrUpdateToolCall(ctx context.Context, messageID, toolCallID, toolName, arguments string) (*model.ChatToolCall, error)
	UpdateToolResult(ctx context.Context, toolCallID, result string, durationMs int) error
}

// chatService 实现
type chatService struct {
	sessionRepo  repository.ChatSessionRepository
	messageRepo  repository.ChatMessageRepository
	toolCallRepo repository.ChatToolCallRepository
}

// NewChatService 创建服务实例
func NewChatService(
	sessionRepo repository.ChatSessionRepository,
	messageRepo repository.ChatMessageRepository,
	toolCallRepo repository.ChatToolCallRepository,
) ChatService {
	return &chatService{
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		toolCallRepo: toolCallRepo,
	}
}

// generateID 生成唯一ID
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// CreateSession 创建会话
func (s *chatService) CreateSession(ctx context.Context, repoID uint) (*model.ChatSession, error) {
	session := &model.ChatSession{
		SessionID: generateID("sess"),
		RepoID:    repoID,
		Title:     "新对话",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return session, nil
}

// GetSession 获取会话
func (s *chatService) GetSession(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	return s.sessionRepo.GetBySessionID(ctx, sessionID)
}

// ListSessions 获取会话列表
func (s *chatService) ListSessions(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error) {
	return s.sessionRepo.ListByRepoID(ctx, repoID, page, pageSize)
}

// ListPublicSessions 获取公开会话列表
func (s *chatService) ListPublicSessions(ctx context.Context, repoID uint, page, pageSize int) ([]*model.ChatSession, int64, error) {
	return s.sessionRepo.ListPublicByRepoID(ctx, repoID, page, pageSize)
}

// UpdateSessionVisibility 更新会话可见性
func (s *chatService) UpdateSessionVisibility(ctx context.Context, sessionID, visibility string) error {
	return s.sessionRepo.UpdateVisibility(ctx, sessionID, visibility)
}

// UpdateMessageCount 更新消息数量
func (s *chatService) UpdateMessageCount(ctx context.Context, sessionID string) error {
	count, err := s.messageRepo.CountBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.sessionRepo.UpdateMessageCount(ctx, sessionID, int(count))
}

// DeleteSession 删除会话
func (s *chatService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// UpdateSessionTitle 更新会话标题
func (s *chatService) UpdateSessionTitle(ctx context.Context, sessionID, title string) error {
	return s.sessionRepo.UpdateTitle(ctx, sessionID, title)
}

// CreateUserMessage 创建用户消息
func (s *chatService) CreateUserMessage(ctx context.Context, sessionID, content string) (*model.ChatMessage, error) {
	message := &model.ChatMessage{
		SessionID:   sessionID,
		MessageID:   generateID("msg"),
		Role:        "user",
		Content:     content,
		ContentType: "text",
		Status:      "completed",
		CreatedAt:   time.Now(),
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("创建用户消息失败: %w", err)
	}

	return message, nil
}

// CreateAssistantMessage 创建AI消息（初始状态streaming）
func (s *chatService) CreateAssistantMessage(ctx context.Context, sessionID string) (*model.ChatMessage, error) {
	message := &model.ChatMessage{
		SessionID:   sessionID,
		MessageID:   generateID("msg"),
		Role:        "assistant",
		Content:     "",
		ContentType: "text",
		Status:      "streaming",
		CreatedAt:   time.Now(),
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("创建AI消息失败: %w", err)
	}

	return message, nil
}

// GetMessage 获取消息
func (s *chatService) GetMessage(ctx context.Context, messageID string) (*model.ChatMessage, error) {
	return s.messageRepo.GetByMessageID(ctx, messageID)
}

// ListMessages 获取消息列表
func (s *chatService) ListMessages(ctx context.Context, sessionID string, limit int, beforeID *string) ([]*model.ChatMessage, error) {
	return s.messageRepo.ListBySessionID(ctx, sessionID, limit, beforeID)
}

// UpdateMessageContent 更新消息内容
func (s *chatService) UpdateMessageContent(ctx context.Context, messageID, content string) error {
	return s.messageRepo.UpdateContent(ctx, messageID, content)
}

// FinalizeMessage 完成消息
func (s *chatService) FinalizeMessage(ctx context.Context, messageID string, tokenUsed int, status string) error {
	completedAt := time.Now()
	return s.messageRepo.Finalize(ctx, messageID, tokenUsed, status, completedAt)
}

// CountMessages 统计消息数量
func (s *chatService) CountMessages(ctx context.Context, sessionID string) (int64, error) {
	return s.messageRepo.CountBySessionID(ctx, sessionID)
}

// CreateToolCall 创建工具调用记录
func (s *chatService) CreateToolCall(ctx context.Context, messageID, toolCallID, toolName, arguments string) (*model.ChatToolCall, error) {
	now := time.Now()
	toolCall := &model.ChatToolCall{
		MessageID:  messageID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
		Status:     "running",
		StartedAt:  &now,
	}

	if err := s.toolCallRepo.Create(ctx, toolCall); err != nil {
		return nil, fmt.Errorf("创建工具调用记录失败: %w", err)
	}

	return toolCall, nil
}

// CreateOrUpdateToolCall 创建或更新工具调用记录（Upsert）
func (s *chatService) CreateOrUpdateToolCall(ctx context.Context, messageID, toolCallID, toolName, arguments string) (*model.ChatToolCall, error) {
	now := time.Now()
	toolCall := &model.ChatToolCall{
		MessageID:  messageID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
		Status:     "running",
		StartedAt:  &now,
	}

	if err := s.toolCallRepo.CreateOrUpdate(ctx, toolCall); err != nil {
		return nil, fmt.Errorf("创建或更新工具调用记录失败: %w", err)
	}

	return toolCall, nil
}

// UpdateToolResult 更新工具调用结果
func (s *chatService) UpdateToolResult(ctx context.Context, toolCallID, result string, durationMs int) error {
	return s.toolCallRepo.UpdateResult(ctx, toolCallID, result, durationMs)
}
