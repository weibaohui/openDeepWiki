package agents

import "time"

// RouterContext 路由上下文
type RouterContext struct {
	AgentName  string            // 显式指定的 Agent name
	EntryPoint string            // 用户入口（如 "diagnose", "ops"）
	TaskType   string            // 任务类型
	Metadata   map[string]string // 附加元数据
}

// LoadResult 加载结果
type LoadResult struct {
	Agent  *Agent
	Error  error
	Action string // "created", "updated", "failed"
}

// FileEvent 文件事件
type FileEvent struct {
	Type string // "create", "modify", "delete"
	Path string
}

// Now 返回当前时间（用于测试）
var Now = func() time.Time {
	return time.Now()
}
