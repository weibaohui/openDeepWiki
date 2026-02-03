package utils

import (
	"encoding/json"

	"k8s.io/klog/v2"
)

// extractJSON 从文本中提取 JSON 部分
func ExtractJSON(content string) string {
	start := -1
	end := -1
	depth := 0

	for i, ch := range content {
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 && start != -1 {
				end = i + 1
				break
			}
		}
	}

	if start >= 0 && end > start {
		return content[start:end]
	}

	return content
}

func ToJSON(v any) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		klog.Errorf("JSON序列化失败: %v", err)
		return ""
	}
	return string(jsonData)
}

// ExtractMarkdown 从文本中提取 Markdown 内容
// 尝试提取 ```markdown ... ``` 代码块，如果没有代码块则返回原始内容
func ExtractMarkdown(content string) string {
	start := -1
	end := -1
	depth := 0
	inCodeBlock := false
	codeBlockPrefix := "```"

	for i := 0; i < len(content); {
		// 检查是否是代码块开始标记
		if i+3 <= len(content) && content[i:i+3] == codeBlockPrefix {
			if !inCodeBlock {
				// 找到代码块开始
				inCodeBlock = true
				// 跳过 ``` 和可能的 markdown 标识
				j := i + 3
				// 跳过空格和可选的 markdown 标识
				for j < len(content) && (content[j] == ' ' || content[j] == 'm' || content[j] == 'M') {
					j++
				}
				// 跳过换行符
				for j < len(content) && (content[j] == '\r' || content[j] == '\n') {
					j++
				}
				if depth == 0 {
					start = j
				}
				depth++
				i = j
			} else {
				// 找到代码块结束
				depth--
				if depth == 0 && start != -1 {
					end = i
					break
				}
				inCodeBlock = false
				i += 3
			}
		} else {
			i++
		}
	}

	// 如果找到代码块，返回代码块内容
	if start >= 0 && end > start {
		klog.V(6).Infof("[ExtractMarkdown] 提取到 Markdown 代码块，起始位置: %d, 结束位置: %d", start, end)
		return content[start:end]
	}

	// 如果没有找到代码块，返回原始内容
	klog.V(6).Infof("[ExtractMarkdown] 未找到 Markdown 代码块，返回原始内容")
	return content
}
