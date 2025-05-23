package config

import (
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

func GetConfig(c *gin.Context) {
	config, err := service.ConfigService().GetConfig()

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonData(c, config)
}

func UpdateConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	if err := service.ConfigService().UpdateConfig(&config); err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	_ = service.ConfigService().UpdateFlagFromDBConfig()

	// 重新加载AI客户端
	_, _ = service.AIService().ReloadDefaultClient()
	amis.WriteJsonOK(c)
}
