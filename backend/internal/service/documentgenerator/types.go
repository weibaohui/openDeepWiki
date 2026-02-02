package documentgenerator

import (
	"encoding/json"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

// DocumentGenerationResult 文档生成结果
type DocumentGenerationResult struct {
	Content         string `json:"content"`          // 生成的文档内容
	AnalysisSummary string `json:"analysis_summary"` // 分析摘要
}

// ParseDocumentGenerationResult 从 Agent 输出解析文档生成结果
// content: Agent 返回的原始内容
// 返回: 解析后的结果或错误
func ParseDocumentGenerationResult(content string) (*DocumentGenerationResult, error) {
	klog.V(6).Infof("[ParseDocumentGenerationResult] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 尝试从内容中提取 JSON
	jsonStr := utils.ExtractJSON(content)
	if jsonStr == "" {
		klog.Warningf("[ParseDocumentGenerationResult] 未能从内容中提取 JSON")
		return nil, fmt.Errorf("未能从 Agent 输出中提取有效 JSON")
	}

	var result DocumentGenerationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		klog.Errorf("[ParseDocumentGenerationResult] JSON 解析失败: %v", err)
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 校验结果
	if result.Content == "" {
		return nil, fmt.Errorf("生成的文档内容为空")
	}

	klog.V(6).Infof("[ParseDocumentGenerationResult] 解析成功，内容长度: %d", len(result.Content))
	return &result, nil
}
