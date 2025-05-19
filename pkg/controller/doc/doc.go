package doc

import (
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

// Init 处理对指定 Git 仓库的文档服务初始化请求。
// 
// 该函数为 "openDeepWiki" 仓库创建用户上下文，并尝试克隆仓库代码。克隆失败时返回 JSON 格式的错误信息，成功则返回标准 JSON 成功响应。
func Init(c *gin.Context) {

	ctx := amis.GetNewContextWithUser(c)

	docService := service.NewDocService(&models.Repo{
		Name:        "openDeepWiki",
		Description: "",
		RepoType:    "git",
		URL:         "https://github.com/weibaohui/openDeepWiki.git",
		Branch:      "main",
	})
	err := docService.RepoService().Clone(ctx)

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}

// Readme 生成指定 Git 仓库的 README 文档，并以 JSON 格式返回操作结果。
func Readme(c *gin.Context) {

	ctx := amis.GetNewContextWithUser(c)

	docService := service.NewDocService(&models.Repo{
		Name:        "openDeepWiki",
		Description: "",
		RepoType:    "git",
		URL:         "https://github.com/weibaohui/openDeepWiki.git",
		Branch:      "main",
	})
	err := docService.ReadmeService().Generate(ctx)

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}
