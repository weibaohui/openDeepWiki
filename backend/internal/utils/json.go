package utils

import (
	"encoding/json"

	"k8s.io/klog/v2"
)

func ToJSON(v any) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		klog.Errorf("JSON序列化失败: %v", err)
		return ""
	}
	return string(jsonData)
}
