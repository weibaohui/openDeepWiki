package statemachine

import (
	"fmt"
	"k8s.io/klog/v2"
)

// TaskStatus 定义任务的所有可能状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 未运行（初始态/重置态）
	TaskStatusQueued    TaskStatus = "queued"    // 已入队等待
	TaskStatusRunning   TaskStatus = "running"   // 正在执行
	TaskStatusSucceeded TaskStatus = "succeeded" // 执行成功（替代completed更语义化）
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCanceled  TaskStatus = "canceled"  // 被取消
)

// TaskTransition 定义任务状态迁移
type TaskTransition struct {
	From TaskStatus
	To   TaskStatus
}

// TaskStateMachine 任务状态机
type TaskStateMachine struct {
	// 定义所有合法的状态迁移
	allowedTransitions map[TaskTransition]bool
}

// NewTaskStateMachine 创建新的任务状态机
func NewTaskStateMachine() *TaskStateMachine {
	sm := &TaskStateMachine{
		allowedTransitions: make(map[TaskTransition]bool),
	}

	// 定义合法的状态迁移路径
	// pending -> queued -> running -> succeeded/failed
	// running -> failed（超时/异常）
	// queued/running -> canceled（用户取消）
	// failed/succeeded/canceled -> pending（reset）
	transitions := []TaskTransition{
		// 正常执行流程
		{TaskStatusPending, TaskStatusQueued},
		{TaskStatusQueued, TaskStatusRunning},
		{TaskStatusRunning, TaskStatusSucceeded},
		{TaskStatusRunning, TaskStatusFailed},

		// 失败重置流程
		{TaskStatusFailed, TaskStatusPending},
		{TaskStatusSucceeded, TaskStatusPending},
		{TaskStatusCanceled, TaskStatusPending},

		// 取消流程
		{TaskStatusQueued, TaskStatusCanceled},
		{TaskStatusRunning, TaskStatusCanceled},
	}

	for _, t := range transitions {
		sm.allowedTransitions[t] = true
	}

	return sm
}

// CanTransition 检查状态迁移是否合法
func (sm *TaskStateMachine) CanTransition(from, to TaskStatus) bool {
	if from == to {
		return false // 不允许状态不变
	}
	return sm.allowedTransitions[TaskTransition{From: from, To: to}]
}

// ValidateTransition 验证状态迁移并返回错误
func (sm *TaskStateMachine) ValidateTransition(from, to TaskStatus) error {
	if !sm.CanTransition(from, to) {
		return &InvalidStateTransitionError{
			From: string(from),
			To:   string(to),
		}
	}
	return nil
}

// Transition 执行状态迁移（带日志）
func (sm *TaskStateMachine) Transition(from, to TaskStatus, taskID uint) error {
	if err := sm.ValidateTransition(from, to); err != nil {
		klog.V(6).Infof("任务状态迁移被拒绝: taskID=%d, %s -> %s, error=%v",
			taskID, from, to, err)
		return err
	}

	klog.V(6).Infof("任务状态迁移成功: taskID=%d, %s -> %s", taskID, from, to)
	return nil
}

// InvalidStateTransitionError 无效的状态迁移错误
type InvalidStateTransitionError struct {
	From string
	To   string
}

func (e *InvalidStateTransitionError) Error() string {
	return fmt.Sprintf("invalid task state transition: %s -> %s", e.From, e.To)
}

// IsTerminal 判断状态是否为终止态（不能再迁移）
func IsTerminal(status TaskStatus) bool {
	return status == TaskStatusSucceeded || status == TaskStatusFailed || status == TaskStatusCanceled
}

// IsRunning 判断任务是否在运行中（包括queued和running）
func IsRunning(status TaskStatus) bool {
	return status == TaskStatusQueued || status == TaskStatusRunning
}
