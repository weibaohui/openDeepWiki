package chatdoc

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
