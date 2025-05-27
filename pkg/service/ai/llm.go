package ai

import (
	"context"

	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

// CallLLM 封装统一大模型调用
func CallLLM(ctx context.Context, prompt string) (string, error) {
	client, err := service.AIService().DefaultClient()
	if err != nil {
		klog.Errorf("获取默认AI客户端失败: %v", err)
		return "", err
	}
	return client.GetCompletion(ctx, prompt)
}
