package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
)

// RequireAdmin 检查用户是否为管理员的中间件
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role := amis.GetLoginUser(c)
		if role == "" {
			role = "guest"
		}

		roles := strings.Split(role, ",")
		if !slices.Contains(roles, constants.RoleAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}
