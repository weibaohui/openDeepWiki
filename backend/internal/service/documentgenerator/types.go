package documentgenerator

import (
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// DocumentGenerationResult 文档生成结果
type DocumentGenerationResult struct {
	Content         string `json:"content"`          // 生成的文档内容（Markdown）
	AnalysisSummary string `json:"analysis_summary"` // 分析摘要
}

// ParseDocumentGenerationResult 从 Agent 输出解析文档生成结果
// content: Agent 返回的原始内容
// 返回: 解析后的结果或错误
func ParseDocumentGenerationResult(content string) (*DocumentGenerationResult, error) {
	klog.V(6).Infof("[ParseDocumentGenerationResult] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 从内容中提取 Markdown
	markdownContent := utils.ExtractMarkdown(content)
	if markdownContent == "" {
		klog.Warningf("[ParseDocumentGenerationResult] 未能从内容中提取 Markdown")
		return nil, fmt.Errorf("未能从 Agent 输出中提取有效 Markdown")
	}

	// 校验结果
	if len(markdownContent) < 10 { // 至少要有一些内容
		return nil, fmt.Errorf("提取的 Markdown 内容过短")
	}

	result := &DocumentGenerationResult{
		Content:         markdownContent,
		AnalysisSummary: "文档生成完成",
	}

	klog.V(6).Infof("[ParseDocumentGenerationResult] 解析成功，内容长度: %d", len(result.Content))
	return result, nil
}
