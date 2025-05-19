package repo

import (
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
)

// ListRepoHandler 获取仓库列表
func ListRepoHandler(c *gin.Context) {
	params := dao.BuildParams(c)
	m := &models.Repo{}

	repos, total, err := m.List(params)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonListWithTotal(c, total, repos)
}

// CreateOrUpdateRepoHandler 新建或更新仓库
func CreateOrUpdateRepoHandler(c *gin.Context) {
	params := dao.BuildParams(c)
	m := &models.Repo{}

	if err := c.ShouldBindJSON(m); err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	err := m.Save(params)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}

// DeleteRepoHandler 删除仓库
func DeleteRepoHandler(c *gin.Context) {
	ids := c.Param("ids")
	params := dao.BuildParams(c)
	m := &models.Repo{}

	err := m.Delete(params, ids)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonOK(c)
}
