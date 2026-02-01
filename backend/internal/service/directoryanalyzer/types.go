// Package directoryanalyzer 目录分析任务生成服务
// 基于 Eino ADK 实现，用于分析代码目录并动态生成分析任务
package directoryanalyzer

import (
	"encoding/json"
	"fmt"

	"github.com/opendeepwiki/backend/internal/utils"
	"k8s.io/klog/v2"
)

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

// ParseTaskGenerationResult 从 Agent 输出解析任务生成结果
// content: Agent 返回的原始内容
// 返回: 解析后的结果或错误
func ParseTaskGenerationResult(content string) (*TaskGenerationResult, error) {
	klog.V(6).Infof("[ParseTaskGenerationResult] 开始解析 Agent 输出，内容长度: %d", len(content))

	// 尝试从内容中提取 JSON
	jsonStr := utils.ExtractJSON(content)
	if jsonStr == "" {
		klog.Warningf("[ParseTaskGenerationResult] 未能从内容中提取 JSON")
		return nil, fmt.Errorf("未能从 Agent 输出中提取有效 JSON")
	}

	var result TaskGenerationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		klog.Errorf("[ParseTaskGenerationResult] JSON 解析失败: %v", err)
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 校验结果
	if err := result.Validate(); err != nil {
		klog.Warningf("[ParseTaskGenerationResult] 结果校验失败: %v", err)
		return nil, err
	}

	klog.V(6).Infof("[ParseTaskGenerationResult] 解析成功，任务数: %d", len(result.Tasks))
	return &result, nil
}

// Validate 验证任务生成结果的有效性
func (r *TaskGenerationResult) Validate() error {
	if len(r.Tasks) == 0 {
		return fmt.Errorf("任务列表为空")
	}

	// 检查是否包含 overview
	hasOverview := false
	seenTypes := make(map[string]bool)
	seenOrders := make(map[int]bool)

	for i, task := range r.Tasks {
		// 检查 type 是否为空
		if task.Type == "" {
			return fmt.Errorf("第 %d 个任务 type 为空", i+1)
		}

		// 检查 type 是否唯一
		if seenTypes[task.Type] {
			return fmt.Errorf("任务 type 重复: %s", task.Type)
		}
		seenTypes[task.Type] = true

		// 检查 sort_order 是否冲突
		if seenOrders[task.SortOrder] {
			return fmt.Errorf("sort_order 重复: %d", task.SortOrder)
		}
		seenOrders[task.SortOrder] = true

		// 检查 overview
		if task.Type == TaskTypeOverview {
			hasOverview = true
		}
	}

	if !hasOverview {
		return fmt.Errorf("缺少必需的 overview 任务")
	}

	return nil
}

// SortTasks 按 SortOrder 对任务排序
func (r *TaskGenerationResult) SortTasks() {
	// 使用冒泡排序（任务数通常较少）
	n := len(r.Tasks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if r.Tasks[j].SortOrder > r.Tasks[j+1].SortOrder {
				r.Tasks[j], r.Tasks[j+1] = r.Tasks[j+1], r.Tasks[j]
			}
		}
	}
}

// NormalizeSortOrder 规范化排序顺序，确保从 1 开始连续递增
func (r *TaskGenerationResult) NormalizeSortOrder() {
	r.SortTasks()
	for i := range r.Tasks {
		r.Tasks[i].SortOrder = i + 1
	}
}
