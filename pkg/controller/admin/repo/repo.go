package repo

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

// ListRepoHandler 获取仓库列表
func ListRepoHandler(c *gin.Context) {
	db := dao.DB()
	repos, err := service.ListRepo(c, db)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	amis.WriteJsonData(c, repos)
}

// CreateOrUpdateRepoHandler 新建或更新仓库
func CreateOrUpdateRepoHandler(c *gin.Context) {
	db := dao.DB()
	var repo models.Repo
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if repo.ID == 0 {
		err := service.CreateRepo(c, db, &repo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		err := service.UpdateRepo(c, db, &repo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": 0})
}

// DeleteRepoHandler 删除仓库（支持批量）
func DeleteRepoHandler(c *gin.Context) {
	db := dao.DB()
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, id := range req.IDs {
		_ = service.DeleteRepo(c, db, id)
	}
	c.JSON(http.StatusOK, gin.H{"status": 0})
}

// GetRepoHandler 获取单个仓库
func GetRepoHandler(c *gin.Context) {
	db := dao.DB()
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	repo, err := service.GetRepo(c, db, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, repo)
}
