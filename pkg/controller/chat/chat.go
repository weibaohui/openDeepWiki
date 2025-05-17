package chat

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils/amis"
	"github.com/weibaohui/openDeepWiki/pkg/controller/sse"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

type ResourceData struct {
	Data string `form:"data"`
	// AnyQuestion 任意提问
	Question string `form:"question"`
}

func handleRequest(c *gin.Context, promptFunc func(data interface{}) string) {
	if !service.AIService().IsEnabled() {
		amis.WriteJsonData(c, gin.H{
			"result": "请先配置开启ChatGPT功能",
		})
		return
	}

	var data ResourceData
	err := c.ShouldBindQuery(&data)
	if err != nil {
		amis.WriteJsonError(c, err)
		return
	}

	ctxInst := amis.GetContextWithUser(c)

	prompt := promptFunc(data)

	stream, err := service.ChatService().GetChatStream(ctxInst, prompt)
	if err != nil {
		klog.V(2).Infof("Error Stream chat request:%v\n\n", err)
		return
	}
	sse.WriteWebSocketChatCompletionStream(c, stream)
}

func AnySelection(c *gin.Context) {
	handleRequest(c, func(data interface{}) string {
		d := data.(ResourceData)
		return fmt.Sprintf(
			`
		\n请你详细解释下面的文字： %s 。
		\n注意：
		\n0、使用中文进行回答。
		\n1、你我之间只进行这一轮交互，后面不要再问问题了。
		\n2、请你在给出答案前反思下回答是否逻辑正确，如有问题请先修正，再返回。回答要直接，不要加入上下衔接、开篇语气词、结尾语气词等啰嗦的信息。
		\n3、请不要向我提问，也不要向我确认信息，请不要让我检查markdown格式，不要让我确认markdown格式是否正确`,
			d.Question)
	})
}
