package chatdoc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	model "github.com/weibaohui/openDeepWiki/pkg/models/chatdoc"
	"github.com/weibaohui/openDeepWiki/pkg/service/chatdoc"
)

var chatDocService = chatdoc.NewChatDocService()

func StartSession(c *gin.Context) {
	var req struct {
		InitialTask string `json:"initial_task" form:"initial_task"`
	}
	_ = c.ShouldBind(&req)
	session := chatDocService.StartSession(c.Request.Context(), req.InitialTask)
	c.JSON(http.StatusOK, session)
}

func ExecuteTask(c *gin.Context) {
	var req struct {
		Session model.ChatDocSession `json:"session"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := chatDocService.ExecuteTask(c.Request.Context(), &req.Session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result, "session": req.Session})
}

func StartWorkflow(c *gin.Context) {
	// var req struct {
	// 	InitialContent string `json:"initial_content" form:"initial_content"`
	// }
	// if err := c.ShouldBind(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
	err := chatdoc.StartWorkflow("请编写一个readme文档")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "workflow started"})
}
