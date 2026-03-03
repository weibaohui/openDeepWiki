package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/adkagents"
	"github.com/weibaohui/opendeepwiki/backend/internal/service"
	"k8s.io/klog/v2"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域，生产环境应该配置具体域名
	},
}

// ChatHandler 对话处理器
type ChatHandler struct {
	chatService  service.ChatService
	repoService  *service.RepositoryService
	hub          *ChatHub
	agentFactory *adkagents.AgentFactory
}

// NewChatHandler 创建处理器
func NewChatHandler(chatService service.ChatService, repoService *service.RepositoryService, agentFactory *adkagents.AgentFactory) *ChatHandler {
	return &ChatHandler{
		chatService:  chatService,
		repoService:  repoService,
		hub:          NewChatHub(),
		agentFactory: agentFactory,
	}
}

// GetHub 获取Hub（用于启动）
func (h *ChatHandler) GetHub() *ChatHub {
	return h.hub
}

// RegisterRoutes 注册路由
func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup) {
	chat := r.Group("/repositories/:id/chat")
	{
		// 会话管理
		chat.POST("/sessions", h.CreateSession)
		chat.GET("/sessions", h.ListSessions)
		chat.GET("/sessions/:session_id", h.GetSession)
		chat.DELETE("/sessions/:session_id", h.DeleteSession)

		// WebSocket
		chat.GET("/sessions/:session_id/stream", h.WebSocket)
	}
}

// CreateSession 创建会话
func (h *ChatHandler) CreateSession(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	session, err := h.chatService.CreateSession(c.Request.Context(), uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ListSessions 获取会话列表
func (h *ChatHandler) ListSessions(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	sessions, total, err := h.chatService.ListSessions(c.Request.Context(), uint(repoID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     sessions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSession 获取会话详情
func (h *ChatHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 获取消息列表（默认10条）
	messages, err := h.chatService.ListMessages(c.Request.Context(), sessionID, 10, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  session,
		"messages": messages,
	})
}

// DeleteSession 删除会话
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	if err := h.chatService.DeleteSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// WebSocket WebSocket连接
func (h *ChatHandler) WebSocket(c *gin.Context) {
	repoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的仓库ID"})
		return
	}

	sessionID := c.Param("session_id")

	// 验证会话存在且属于该仓库
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if session.RepoID != uint(repoID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "会话不属于该仓库"})
		return
	}

	// 升级WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket升级失败"})
		return
	}

	// 创建客户端
	client := &Client{
		hub:       h.hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		sessionID: sessionID,
		repoID:    uint(repoID),
		stopChan:  make(chan struct{}),
	}

	// 注册到Hub
	h.hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump(h)
}

// Client 表示一个WebSocket客户端
type Client struct {
	hub       *ChatHub
	conn      *websocket.Conn
	send      chan []byte
	sessionID string
	repoID    uint
	stopChan  chan struct{}
	mu        sync.Mutex
	closed    bool // 标记连接是否已关闭
}

// readPump 读取消息
func (c *Client) readPump(h *ChatHandler) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		// 解析消息
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("INVALID_MESSAGE", "消息格式错误")
			continue
		}

		// 处理消息
		switch msg.Type {
		case "message":
			h.handleMessage(c, &msg)
		case "stop":
			h.handleStop(c)
		case "ping":
			c.sendPong()
		default:
			c.sendError("UNKNOWN_TYPE", "未知的消息类型")
		}
	}
}

// writePump 写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError 发送错误消息（线程安全）
func (c *Client) sendError(code, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	event := ServerMessage{
		Type:      "error",
		ID:        generateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Payload: ErrorPayload{
			Code:      code,
			Message:   message,
			Retryable: false,
		},
	}
	data, _ := json.Marshal(event)
	select {
	case c.send <- data:
	default:
	}
}

// sendPong 发送pong（线程安全）
func (c *Client) sendPong() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	event := ServerMessage{
		Type:      "pong",
		ID:        generateEventID(),
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(event)
	select {
	case c.send <- data:
	default:
	}
}

// handleMessage 处理用户消息
func (h *ChatHandler) handleMessage(client *Client, msg *ClientMessage) {
	ctx := context.Background()

	// 检查消息数量限制
	count, err := h.chatService.CountMessages(ctx, client.sessionID)
	if err != nil {
		client.sendError("INTERNAL_ERROR", "检查消息数量失败")
		return
	}

	if count >= 1000 {
		client.sendError("MESSAGE_LIMIT", "会话消息数量已达上限")
		return
	}

	// 保存用户消息
	userMsg, err := h.chatService.CreateUserMessage(ctx, client.sessionID, msg.Content)
	if err != nil {
		client.sendError("INTERNAL_ERROR", "保存消息失败")
		return
	}

	// 如果是第一条消息，更新会话标题
	if count == 0 {
		title := msg.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		h.chatService.UpdateSessionTitle(ctx, client.sessionID, title)
	}

	// 启动Agent执行
	go h.runAgent(client, userMsg)
}

// handleStop 处理停止请求
func (h *ChatHandler) handleStop(client *Client) {
	client.mu.Lock()
	defer client.mu.Unlock()

	close(client.stopChan)
	client.stopChan = make(chan struct{})
}

// runAgent 运行Agent
func (h *ChatHandler) runAgent(client *Client, userMsg *model.ChatMessage) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听停止信号
	go func() {
		select {
		case <-client.stopChan:
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	// 获取仓库信息
	var repoInfo string
	if h.repoService != nil {
		repo, err := h.repoService.Get(client.repoID)
		if err == nil && repo != nil {
			repoInfo = fmt.Sprintf("## 当前仓库信息\n- 仓库名称: %s\n- 仓库地址: %s\n- 本地路径: %s\n- 仓库描述: %s\n- 当前分支: %s\n- 当前Commit: %s\n",
				repo.Name, repo.URL, repo.LocalPath, repo.Description, repo.CloneBranch, repo.CloneCommit)
		}
	}

	// 获取 Agent
	agent, err := h.agentFactory.GetAgent("chat_assistant")
	if err != nil {
		client.sendError("AGENT_NOT_FOUND", fmt.Sprintf("无法获取Agent: %v", err))
		return
	}

	// 创建AI消息记录
	assistantMsg, err := h.chatService.CreateAssistantMessage(ctx, client.sessionID)
	if err != nil {
		client.sendError("INTERNAL_ERROR", "创建AI消息失败")
		return
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

	// 获取历史消息
	historyMsgs, err := h.chatService.ListMessages(ctx, client.sessionID, 20, nil)
	if err != nil {
		client.sendError("INTERNAL_ERROR", "获取历史消息失败")
		return
	}

	// 构建ADK消息列表
	var adkMessages []*schema.Message

	// 添加系统消息（仓库信息）
	if repoInfo != "" {
		adkMessages = append(adkMessages, &schema.Message{
			Role:    schema.System,
			Content: repoInfo,
		})
	}

	for _, msg := range historyMsgs {
		if msg.Status == "completed" || msg.Status == "streaming" {
			role := schema.User
			if msg.Role == "assistant" {
				role = schema.Assistant
			}
			adkMessages = append(adkMessages, &schema.Message{
				Role:    role,
				Content: msg.Content,
			})
		}
	}

	// 创建Runner并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	iter := runner.Run(ctx, adkMessages)

	var fullContent string
	var tokenUsed int
	// 追踪已发送的 content，避免重复
	sentContents := make(map[string]bool)

	// 遍历事件流
	for {
		select {
		case <-ctx.Done():
			// 用户取消或超时
			h.chatService.FinalizeMessage(ctx, assistantMsg.MessageID, tokenUsed, "stopped")
			client.sendEvent(ServerMessage{
				Type:      "stopped",
				ID:        generateEventID(),
				Timestamp: time.Now().UnixMilli(),
				Payload: map[string]interface{}{
					"message_id": assistantMsg.MessageID,
					"reason":     "user_cancelled",
				},
			})
			return
		default:
		}

		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			// 执行出错
			h.chatService.FinalizeMessage(ctx, assistantMsg.MessageID, tokenUsed, "error")
			client.sendError("AGENT_ERROR", event.Err.Error())
			return
		}

		// 处理输出事件
		if event.Output != nil && event.Output.MessageOutput != nil {
			// 跳过 role="tool" 的系统消息（不发送给前端，也不存数据库）
			if event.Output.MessageOutput.Role == "tool" {
				klog.V(6).Info("跳过 role=tool 的系统消息，不发送给前端")
				continue
			}

			content := event.Output.MessageOutput.Message.Content
			// 对非 final 开头的 assistant 消息，自动添加 thinking 包裹
			if event.Output.MessageOutput.Role == "assistant" && len(strings.TrimSpace(content)) > 0 && !strings.HasPrefix(content, "<final>") && !strings.HasPrefix(content, "<thinking>") {
				content = "<thinking>" + content + "</thinking>"
			}
			if content != "" {
				// 检查是否已发送过相同内容，避免重复
				if !sentContents[content] {
					sentContents[content] = true
					// 发送内容增量
					client.sendEvent(ServerMessage{
						Type:      "content_delta",
						ID:        assistantMsg.MessageID,
						Timestamp: time.Now().UnixMilli(),
						Payload: map[string]interface{}{
							"message_id": assistantMsg.MessageID,
							"delta":      content,
						},
					})
					fullContent += content
					// 更新数据库中的消息内容
					h.chatService.UpdateMessageContent(ctx, assistantMsg.MessageID, fullContent)
				}
				// 发送后清空去重 map，避免无限增长
				sentContents = make(map[string]bool)
			}

			// 处理工具调用
			if len(event.Output.MessageOutput.Message.ToolCalls) > 0 {
				for _, tc := range event.Output.MessageOutput.Message.ToolCalls {
					// 发送工具调用事件
					client.sendEvent(ServerMessage{
						Type:      "tool_call",
						ID:        generateEventID(),
						Timestamp: time.Now().UnixMilli(),
						Payload: map[string]interface{}{
							"tool_call_id": tc.ID,
							"tool_name":    tc.Function.Name,
							"arguments":    tc.Function.Arguments,
						},
					})

					// 保存工具调用到数据库
					h.chatService.CreateOrUpdateToolCall(ctx, assistantMsg.MessageID, tc.ID, tc.Function.Name, tc.Function.Arguments)
				}
			}
		}

		// 检查是否退出
		if event.Action != nil && event.Action.Exit {
			break
		}
	}

	// 完成消息
	h.chatService.FinalizeMessage(ctx, assistantMsg.MessageID, tokenUsed, "completed")

}

// sendEvent 发送事件（线程安全）
func (c *Client) sendEvent(event ServerMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return // 连接已关闭，不发送
	}

	data, _ := json.Marshal(event)
	select {
	case c.send <- data:
	default:
		// 发送通道已满，丢弃消息
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// ClientMessage 客户端消息
type ClientMessage struct {
	Type    string `json:"type"`              // message, stop, ping
	Content string `json:"content,omitempty"` // type=message时使用
	ID      string `json:"id"`
}

// ServerMessage 服务端消息
type ServerMessage struct {
	Type      string      `json:"type"`
	ID        string      `json:"id"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`
}

// ErrorPayload 错误载荷
type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
