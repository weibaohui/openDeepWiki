package mcpservers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/weibaohui/openDeepWiki/pkg/mcpservers/filesystem"
)

func NewMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"openDeepWiki MCP Server",
		"1.0",
		server.WithResourceCapabilities(true, true),
	)
	s, err := filesystem.Register(s, []string{"data"})
	if err != nil {
		return nil
	}
	return s
}
func NewMCPSSEServer() *server.SSEServer {
	s := NewMCPServer()
	SSEOption := []server.SSEOption{
		server.WithStaticBasePath("/mcp"),
	}
	return server.NewSSEServer(s, SSEOption...)
}

// Adapt 将标准的 http.Handler 适配为 Gin 框架可用的处理函数。
func Adapt(fn func() http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler := fn()
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
