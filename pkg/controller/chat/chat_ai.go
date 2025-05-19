package chat

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

// handleAIChat 处理AI聊天的核心逻辑，包括消息处理和响应读取
//
// 参数:
// - c: gin上下文
// - message: 用户输入的消息
// - responseCh: 用于实时返回AI响应的channel
// 返回:
// - chan struct{}: 用于从外部终止处理过程的channel
// - error: 处理过程中的错误
func handleAIChat(ctx context.Context, message string) (io.Reader, error) {
	// 创建一个带有读写功能的管道
	pr, pw := io.Pipe()

	// 启动一个goroutine来处理AI服务的输出
	go func() {
		defer pw.Close()
		// 调用AI服务处理消息，将输出写入管道
		err := service.ChatService().RunOneRound(ctx, message, pw)
		if err != nil {
			klog.Errorf("AI处理消息失败: %v", err)
			return
		}
	}()

	return pr, nil
}

// AIChat 提供与 ChatGPT 及工具集成的交互式对话功能。
//
// 该函数实现了与AI的核心交互逻辑：
// - 检查AI服务是否启用
// - 处理用户输入的消息
// - 调用ChatGPT并集成可用工具
// - 通过for-select循环实时读取并输出AI的响应
// - 返回完整的AI响应结果
//
// 若AI服务未启用或参数绑定失败，将返回相应错误信息。
func AIChat(c *gin.Context) {

	ctx := amis.GetNewContextWithUser(c)

	docService := service.NewDocService(&models.Repo{
		Name:        "openDeepWiki",
		Description: "",
		RepoType:    "public",
		URL:         "github.com/weibaohui/openDeepWiki",
		Branch:      "main",
	})
	err := docService.ReadmeService().Generate(ctx)

	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	amis.WriteJsonOK(c)
}
