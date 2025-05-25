package doc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

// GetLatestLogs 以服务端推送事件（SSE）的方式，将最新日志文件的内容实时流式传输给客户端。
// 若未找到日志文件或发生错误，则返回相应的错误信息并终止请求。
func GetLatestLogs(c *gin.Context) {
	analysisID := c.Param("analysis_id")
	if analysisID == "" {
		amis.WriteJsonError(c, fmt.Errorf("invalid analysis ID"))
		return
	}
	ctx := c.Request.Context()

	docService := service.NewDocServiceWithAnalysisID(analysisID)

	_, err := docService.GetRuntimeFilePath()
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	updates, err := docService.TailFile(ctx)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	// 设置响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)
	// 持续发送文件更新
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updates:
			if !ok {
				// 通道已关闭，发送结束标记
				c.Writer.Write([]byte("end"))
				return
			}
			klog.V(6).Infof("接收到日志更新: %s", update)
			// 直接写入日志更新并添加换行符
			c.SSEvent("message", update)
			c.Writer.Flush()
			klog.V(6).Infof("发送日志更新: %s", update)
		}
	}
}
