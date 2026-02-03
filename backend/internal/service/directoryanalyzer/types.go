// Package directoryanalyzer 目录分析任务生成服务
// 基于 Eino ADK 实现，用于分析代码目录并动态生成分析任务
package directoryanalyzer

// TaskGenerationResult 任务生成结果
type TaskGenerationResult struct {
	Tasks           []GeneratedTask `json:"tasks"`
	AnalysisSummary string          `json:"analysis_summary"`
}

// GeneratedTask 生成的任务定义
// Type 字段不再局限于预定义值，Agent 可根据项目特征自由定义
type GeneratedTask struct {
	Type      string `json:"type"`       // 任务类型标识，如 "security", "performance", "data-model"
	Title     string `json:"title"`      // 任务标题，如 "安全分析"
	SortOrder int    `json:"sort_order"` // 排序顺序
}
