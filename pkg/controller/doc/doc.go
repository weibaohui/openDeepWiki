package doc

import (
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
)

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
