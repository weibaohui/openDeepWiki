package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"gorm.io/gorm"
)

func ToolsList(c *gin.Context) {
	params := dao.BuildParams(c)
	params.PerPage = 10000
	var tool models.MCPTool
	list, _, err := tool.List(params, func(db *gorm.DB) *gorm.DB {
		return db.Order("name asc")
	})
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonList(c, list)
}

func ToolQuickSave(c *gin.Context) {
	id := c.Param("id")
	status := c.Param("status")

	var entity models.MCPTool
	entity.ID = utils.ToUInt(id)

	if status == "true" {
		entity.Enabled = true
	} else {
		entity.Enabled = false
	}
	err := dao.DB().Model(&entity).Select("Enabled").Updates(entity).Error

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonErrorOrOK(c, err)
}
