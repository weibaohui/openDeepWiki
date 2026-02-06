package statemachine

import (
	"fmt"

	"k8s.io/klog/v2"
)

// RepositoryStatus 定义仓库的所有可能状态
type RepositoryStatus string

const (
	RepoStatusPending   RepositoryStatus = "pending"   // 刚创建但还未克隆/未可用
	RepoStatusCloning   RepositoryStatus = "cloning"   // 正在拉取代码
	RepoStatusReady     RepositoryStatus = "ready"     // 可执行任务（至少 clone 成功）
	RepoStatusAnalyzing RepositoryStatus = "analyzing" // 任一任务处于 queued/running
	RepoStatusCompleted RepositoryStatus = "completed" // 所有必选任务 succeeded
	RepoStatusError     RepositoryStatus = "error"     // 至少一个必选任务 failed，且未被修复/重试
)

// RepositoryStateMachine 仓库状态机
type RepositoryStateMachine struct {
	// 定义所有合法的状态迁移
	allowedTransitions map[RepositoryTransition]bool
}

// RepositoryTransition 定义仓库状态迁移
type RepositoryTransition struct {
	From RepositoryStatus
	To   RepositoryStatus
}

// NewRepositoryStateMachine 创建新的仓库状态机
func NewRepositoryStateMachine() *RepositoryStateMachine {
	sm := &RepositoryStateMachine{
		allowedTransitions: make(map[RepositoryTransition]bool),
	}

	// 定义合法的状态迁移路径
	// pending -> cloning -> ready -> analyzing -> completed/error
	transitions := []RepositoryTransition{
		// 克隆流程
		{RepoStatusPending, RepoStatusCloning},
		{RepoStatusReady, RepoStatusCloning},
		{RepoStatusCompleted, RepoStatusCloning},
		{RepoStatusError, RepoStatusCloning},
		{RepoStatusCloning, RepoStatusReady},
		{RepoStatusCloning, RepoStatusError}, // 克隆失败

		// 分析流程
		{RepoStatusReady, RepoStatusAnalyzing},
		{RepoStatusCompleted, RepoStatusAnalyzing}, // 重新分析

		// 分析结果
		{RepoStatusAnalyzing, RepoStatusCompleted},
		{RepoStatusAnalyzing, RepoStatusError},
		{RepoStatusAnalyzing, RepoStatusReady}, // 任务被取消/未完成时回到可执行态

		// 错误恢复
		{RepoStatusError, RepoStatusAnalyzing}, // 重新尝试
		{RepoStatusError, RepoStatusReady},     // 用户手动重置
		{RepoStatusCompleted, RepoStatusReady}, // 重新分析准备
	}

	for _, t := range transitions {
		sm.allowedTransitions[t] = true
	}

	return sm
}

// CanTransition 检查状态迁移是否合法
func (sm *RepositoryStateMachine) CanTransition(from, to RepositoryStatus) bool {
	if from == to {
		return false // 不允许状态不变
	}
	return sm.allowedTransitions[RepositoryTransition{From: from, To: to}]
}

// ValidateTransition 验证状态迁移并返回错误
func (sm *RepositoryStateMachine) ValidateTransition(from, to RepositoryStatus) error {
	if !sm.CanTransition(from, to) {
		return &InvalidRepositoryStateTransitionError{
			From: string(from),
			To:   string(to),
		}
	}
	return nil
}

// Transition 执行状态迁移（带日志）
func (sm *RepositoryStateMachine) Transition(from, to RepositoryStatus, repoID uint) error {
	if err := sm.ValidateTransition(from, to); err != nil {
		klog.V(6).Infof("仓库状态迁移被拒绝: repoID=%d, %s -> %s, error=%v",
			repoID, from, to, err)
		return err
	}

	klog.V(6).Infof("仓库状态迁移成功: repoID=%d, %s -> %s", repoID, from, to)
	return nil
}

// InvalidRepositoryStateTransitionError 无效的仓库状态迁移错误
type InvalidRepositoryStateTransitionError struct {
	From string
	To   string
}

func (e *InvalidRepositoryStateTransitionError) Error() string {
	return fmt.Sprintf("invalid repository state transition: %s -> %s", e.From, e.To)
}

// RepositoryStatusAggregator 仓库状态聚合器
// 根据任务集合状态推导仓库状态
type RepositoryStatusAggregator struct {
	StateMachine *RepositoryStateMachine
}

// NewRepositoryStatusAggregator 创建仓库状态聚合器
func NewRepositoryStatusAggregator() *RepositoryStatusAggregator {
	return &RepositoryStatusAggregator{
		StateMachine: NewRepositoryStateMachine(),
	}
}

// TaskStatusSummary 任务状态汇总
type TaskStatusSummary struct {
	Total     int
	Pending   int
	Queued    int
	Running   int
	Succeeded int
	Failed    int
	Canceled  int
}

// AggregateStatus 根据任务状态聚合仓库状态
// 核心规则：
// 1. 若存在 running/queued：repo = analyzing
// 2. 否则若存在 failed：repo = error
// 3. 否则若所有"必选任务"都 succeeded：repo = completed
// 4. 否则：repo = ready
func (a *RepositoryStatusAggregator) AggregateStatus(
	currentRepoStatus RepositoryStatus,
	summary *TaskStatusSummary,
	repoID uint,
) (RepositoryStatus, error) {
	newStatus := currentRepoStatus

	// 如果还在克隆中或准备中（尚未开始分析），保持原状态
	if currentRepoStatus == RepoStatusPending || currentRepoStatus == RepoStatusCloning {
		return currentRepoStatus, nil
	}

	// 规则1：若存在 running/queued，状态为 analyzing
	if summary.Queued > 0 || summary.Running > 0 {
		newStatus = RepoStatusAnalyzing
	} else if summary.Failed > 0 {
		// 规则2：否则若存在 failed，状态为 error
		newStatus = RepoStatusError
	} else if summary.Succeeded == summary.Total && summary.Total > 0 {
		// 规则3：否则若所有任务都 succeeded，状态为 completed
		newStatus = RepoStatusCompleted
	} else {
		// 规则4：否则状态为 ready
		newStatus = RepoStatusReady
	}

	// 如果状态没有变化，直接返回
	if newStatus == currentRepoStatus {
		return currentRepoStatus, nil
	}

	// 验证状态迁移是否合法
	if err := a.StateMachine.ValidateTransition(currentRepoStatus, newStatus); err != nil {
		klog.Warningf("仓库状态聚合产生了不合法的迁移: repoID=%d, %s -> %s, error=%v",
			repoID, currentRepoStatus, newStatus, err)
		return currentRepoStatus, err
	}

	klog.V(6).Infof("仓库状态聚合: repoID=%d, %s -> %s (summary: total=%d, pending=%d, queued=%d, running=%d, succeeded=%d, failed=%d, canceled=%d)",
		repoID, currentRepoStatus, newStatus,
		summary.Total, summary.Pending, summary.Queued, summary.Running, summary.Succeeded, summary.Failed, summary.Canceled)

	return newStatus, nil
}

// IsRepoTerminal 判断仓库状态是否为终止态
func IsRepoTerminal(status RepositoryStatus) bool {
	return status == RepoStatusCompleted || status == RepoStatusError
}

// CanExecuteTasks 判断仓库是否可以执行任务
func CanExecuteTasks(status RepositoryStatus) bool {
	return status == RepoStatusReady || status == RepoStatusCompleted || status == RepoStatusError
}
