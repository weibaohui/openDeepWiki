package doc

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

// GetLatestLogs 获取最新的日志文件内容并通过HTTP流式传输
func GetLatestLogs(c *gin.Context) {
	ctx := c.Request.Context()

	docService := service.NewDocService(testRepo)

	// 获取最新的日志文件名
	filename, err := docService.GetLatestLogFile(ctx)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	if filename == "" {
		amis.WriteJsonError(c, fmt.Errorf("no log file found"))
		return
	}

	// 设置响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	// 开始监控文件变化
	updates, err := docService.TailFile(ctx, filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

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
