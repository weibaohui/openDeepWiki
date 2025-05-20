package doc

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/models"
)

// GetAnalysisResults 获取特定分析ID的所有结果文档
func GetAnalysisResults(c *gin.Context) {

	analysisID := c.Param("id")
	if analysisID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid analysis ID"})
		return
	}

	analysisIDInt, err := strconv.Atoi(analysisID)

	result := &models.AnalysisResult{}
	params := dao.BuildParams(c)
	results, total, err := result.GetByAnalysisID(params, uint(analysisIDInt))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
	})
}
