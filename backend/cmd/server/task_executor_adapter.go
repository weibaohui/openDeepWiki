package main

import (
	"context"

	"github.com/opendeepwiki/backend/internal/service"
)

// taskExecutorAdapter 将TaskService适配为TaskExecutor接口
// 避免orchestrator和service之间的循环依赖
type taskExecutorAdapter struct {
	taskService *service.TaskService
}

// ExecuteTask 执行任务
// 实现orchestrator.TaskExecutor接口
func (a *taskExecutorAdapter) ExecuteTask(ctx context.Context, taskID uint) error {
	return a.taskService.Run(ctx, taskID)
}
