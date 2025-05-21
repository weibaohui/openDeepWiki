package doc

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

var testRepo = &models.Repo{
	ID:          3,
	Name:        "openDeepWiki",
	Description: "",
	RepoType:    "git",
	URL:         "https://github.com/weibaohui/openDeepWiki.git",
	Branch:      "main",
}

// Init 处理对指定 Git 仓库的文档服务初始化请求。
//
// Init 为 "openDeepWiki" 仓库创建用户上下文并尝试克隆仓库代码，克隆失败时返回 JSON 错误信息，成功则返回标准 JSON 成功响应。
func Init(c *gin.Context) {

	ctx := amis.GetNewContextWithUser(c)

	docService := service.NewDocService(testRepo)
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

	docService := service.NewDocService(testRepo)

	// 创建新的文档解读实例
	analysis, err := docService.AnalysisService().Create(ctx)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	// 生成README文档
	err = docService.ReadmeService().Generate(ctx, analysis)
	if err != nil {
		_ = docService.AnalysisService().UpdateStatus(ctx, analysis, "failed", "", err)
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}

// GetAnalysisHistory 获取代码仓库的分析历史
func GetAnalysisHistory(c *gin.Context) {
	repoID := c.Param("repo_id")
	if repoID == "" {
		amis.WriteJsonError(c, fmt.Errorf("invalid repository ID"))
		return
	}
	repoIDInt, err := strconv.Atoi(repoID)
	if err != nil {
		amis.WriteJsonError(c, fmt.Errorf("invalid repository ID"))
		return
	}

	analysis := &models.DocAnalysis{}
	params := dao.BuildParams(c)
	results, total, err := analysis.GetByRepoID(params, uint(repoIDInt))
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonListWithTotal(c, total, results)
}
