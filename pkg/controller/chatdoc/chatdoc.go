package chatdoc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/service/chatdoc"
)

// 只保留新版多智能体协作入口
func StartWorkflow(c *gin.Context) {
	var req struct {
		InitialContent string `json:"initial_content" form:"initial_content"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := chatdoc.StartWorkflow(req.InitialContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "workflow started"})
}
