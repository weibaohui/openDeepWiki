package repo

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册仓库相关路由
func RegisterRoutes(router *gin.RouterGroup) {
	repo := router.Group("/repo")
	{
		repo.GET("/list", ListRepoHandler)            // 获取仓库列表
		repo.POST("/save", CreateOrUpdateRepoHandler) // 创建或更新仓库
		repo.POST("/delete/:ids", DeleteRepoHandler)  // 删除仓库
	}
}
