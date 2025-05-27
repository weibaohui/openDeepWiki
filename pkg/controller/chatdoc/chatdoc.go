package chatdoc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/service/chatdoc"
)

var chatDocService = chatdoc.NewChatDocService()

func StartSession(c *gin.Context) {
	session := chatDocService.StartSession(c.Request.Context())
	c.JSON(http.StatusOK, session)
}
