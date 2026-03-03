package handler

// ChatHub 管理所有WebSocket连接
type ChatHub struct {
	// 注册通道
	register chan *Client
	// 注销通道
	unregister chan *Client
	// 客户端映射 sessionID -> Client
	clients map[string]*Client
}

// NewChatHub 创建ChatHub
func NewChatHub() *ChatHub {
	return &ChatHub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
	}
}

// Run 启动Hub
func (h *ChatHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.sessionID] = client

		case client := <-h.unregister:
			if _, ok := h.clients[client.sessionID]; ok {
				delete(h.clients, client.sessionID)
				// 先设置关闭标志，再关闭通道
				client.mu.Lock()
				if !client.closed {
					client.closed = true
					close(client.send)
				}
				client.mu.Unlock()
			}
		}
	}
}
