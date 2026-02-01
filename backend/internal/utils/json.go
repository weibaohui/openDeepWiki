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
