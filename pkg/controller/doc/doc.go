package doc

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

// Init 处理对指定 Git 仓库的文档服务初始化请求。
func Init(c *gin.Context) {
	repoID := c.Param("repo_id")

	ctx := amis.GetNewContextWithUser(c)

	docService := service.NewDocServiceWithRepoID(repoID)
	err := docService.RepoService().Clone(ctx)

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}

func Analysis(c *gin.Context) {
	repoID := c.Param("repo_id")
	if repoID == "" {
		amis.WriteJsonError(c, fmt.Errorf("invalid repository ID"))
		return
	}
	ctx := amis.GetNewContextWithUser(c)

	// 初始化文档服务
	docService := service.NewDocServiceWithRepoID(repoID)

	// 创建新的文档解读实例
	analysis, err := docService.AnalysisService().Create(ctx)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}
	docService.SetAnalysis(analysis)
	go func() {
		// 生成README文档
		err = docService.ReadmeService().Generate(ctx)
		if err != nil {
			_ = docService.AnalysisService().UpdateStatus(ctx, analysis, "failed", "", err)
			return
		}
	}()

	amis.WriteJsonOK(c)
}

// GetAnalysisHistory 获取代码仓库的分析历史
func GetAnalysisHistory(c *gin.Context) {
	repoID := c.Param("repo_id")
	if repoID == "" {
		amis.WriteJsonError(c, fmt.Errorf("invalid repository ID"))
		return
	}
	repoIDInt, err := utils.StringToUintID(repoID)
	if err != nil {
		amis.WriteJsonError(c, fmt.Errorf("invalid repository ID"))
		return
	}

	analysis := &models.DocAnalysis{}
	params := dao.BuildParams(c)
	results, total, err := analysis.GetByRepoID(params, repoIDInt)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonListWithTotal(c, total, results)
}
