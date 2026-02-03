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

// 预定义任务类型参考（Agent 可参考，但不局限于这些）
const (
	TaskTypeOverview     = "overview"      // 项目概览
	TaskTypeArchitecture = "architecture"  // 架构分析
	TaskTypeAPI          = "api"           // 核心接口
	TaskTypeBusinessFlow = "business-flow" // 业务流程
	TaskTypeDeployment   = "deployment"    // 部署配置
	TaskTypeSecurity     = "security"      // 安全分析
	TaskTypePerformance  = "performance"   // 性能分析
	TaskTypeDataModel    = "data-model"    // 数据模型
	TaskTypeTesting      = "testing"       // 测试分析
	TaskTypeDevOps       = "devops"        // DevOps 流程
	TaskTypeFrontend     = "frontend"      // 前端分析
	TaskTypeDatabase     = "database"      // 数据库设计
	TaskTypeCache        = "cache"         // 缓存策略
	TaskTypeMessageQueue = "mq"            // 消息队列
	TaskTypeAuth         = "auth"          // 认证授权
	TaskTypeLogMonitor   = "log-monitor"   // 日志监控
	TaskTypeEventDriven  = "event-driven"  // 事件驱动
	TaskTypeScalability  = "scalability"   // 可扩展性
)
