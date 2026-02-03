package documentgenerator

import (
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// ParseDocumentGenerationResult 从 Agent 输出解析文档生成结果
// content: Agent 返回的原始内容
// 返回: 解析后的结果或错误
func ParseDocumentGenerationResult(content string) (string, error) {
	klog.V(6).Infof("[ParseDocumentGenerationResult] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 从内容中提取 Markdown
	markdownContent := utils.ExtractMarkdown(content)
	if markdownContent == "" {
		klog.Warningf("[ParseDocumentGenerationResult] 未能从内容中提取 Markdown")
		return "", fmt.Errorf("未能从 Agent 输出中提取有效 Markdown")
	}

	// 校验结果
	if len(markdownContent) < 10 { // 至少要有一些内容
		return "", fmt.Errorf("提取的 Markdown 内容过短")
	}

	klog.V(6).Infof("[ParseDocumentGenerationResult] 解析成功，内容长度: %d", len(markdownContent))
	return markdownContent, nil
}
