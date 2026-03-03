package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
)

// AgentExecutor Agent执行器接口
type AgentExecutor interface {
	Execute(ctx context.Context, client *Client, sessionID string, messages []*model.ChatMessage) error
}

// agentExecutor 实现
type agentExecutor struct {
	chatService service.ChatService
}

// NewAgentExecutor 创建执行器
func NewAgentExecutor(chatService service.ChatService) AgentExecutor {
	return &agentExecutor{
		chatService: chatService,
	}
}

// Execute 执行Agent
func (e *agentExecutor) Execute(ctx context.Context, client *Client, sessionID string, messages []*model.ChatMessage) error {
	// 创建AI消息
	assistantMsg, err := e.chatService.CreateAssistantMessage(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("创建AI消息失败: %w", err)
	}

	// 发送assistant_start事件
	client.sendEvent(ServerMessage{
		Type:      "assistant_start",
		ID:        generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"message_id": assistantMsg.MessageID,
		},
	})

	// TODO: 实际调用Eino ADK Agent执行
	// 这里暂时模拟Agent执行过程

	// 模拟思考过程
	client.sendEvent(ServerMessage{
		Type:      "thinking_start",
		ID:        generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"message_id": assistantMsg.MessageID,
		},
	})

	// 检查是否被取消
	select {
	case <-ctx.Done():
		e.handleStopped(client, assistantMsg.MessageID)
		return ctx.Err()
	default:
	}

	// TODO: 实际Agent执行完成后，更新消息状态

	return nil
}

// handleStopped 处理停止
func (e *agentExecutor) handleStopped(client *Client, messageID string) {
	// 更新消息状态为stopped
	e.chatService.FinalizeMessage(context.Background(), messageID, 0, "stopped")

	// 发送stopped事件
	client.sendEvent(ServerMessage{
		Type:      "stopped",
		ID:        generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Payload: map[string]interface{}{
			"message_id": messageID,
			"reason":     "user_request",
		},
	})
}

// ToolCall 工具调用
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallPayload 工具调用载荷
type ToolCallPayload struct {
	ToolCallID string          `json:"tool_call_id"`
	ToolName   string          `json:"tool_name"`
	Arguments  json.RawMessage `json:"arguments"`
}

// ToolResultPayload 工具结果载荷
type ToolResultPayload struct {
	ToolCallID string `json:"tool_call_id"`
	Result     string `json:"result"`
	DurationMs int    `json:"duration_ms"`
}

// ContentDeltaPayload 内容增量载荷
type ContentDeltaPayload struct {
	MessageID string `json:"message_id"`
	Delta     string `json:"delta"`
}

// AssistantEndPayload 回答完成载荷
type AssistantEndPayload struct {
	MessageID    string `json:"message_id"`
	TokenUsed    int    `json:"token_used"`
	FinishReason string `json:"finish_reason"`
}
