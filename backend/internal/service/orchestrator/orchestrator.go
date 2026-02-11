package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"k8s.io/klog/v2"
)

// -----------------------------
// Job 定义
// -----------------------------
type Job struct {
	TaskID     uint
	EnqueuedAt time.Time
	RetryCount int
	MaxRetries int
	Timeout    time.Duration
}

// -----------------------------
// TaskExecutor 接口
// -----------------------------
type TaskExecutor interface {
	ExecuteTask(ctx context.Context, taskID uint) error
}

// -----------------------------
// Orchestrator
// -----------------------------
type Orchestrator struct {
	jobQueue    *jobQueue
	retryQueue  *jobQueue
	retryTicker *time.Ticker

	pool *ants.Pool

	executor TaskExecutor

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	activeCancellations map[uint]context.CancelFunc
	cancelMutex         sync.Mutex
}

// -----------------------------
// 错误定义
// -----------------------------
var (
	ErrOrchestratorStopped = errors.New("orchestrator is stopped")
	ErrQueueFull           = errors.New("job queue is full")
	ErrRepoLocked          = errors.New("repository is locked by another task")
)

// NewTaskJob
// 说明：创建一个新的任务对象，初始化重试次数、最大重试次数与自定义超时
// 参数：taskID 任务ID；repositoryID 仓库ID
// 返回：*Job 初始化后的任务对象
func NewTaskJob(taskID, repositoryID uint) *Job {
	return &Job{
		TaskID:     taskID,
		EnqueuedAt: time.Now(),
		RetryCount: 0,
		MaxRetries: 5,
		Timeout:    30 * time.Minute,
	}
}

// -----------------------------
// 构造函数
// -----------------------------
func NewOrchestrator(maxWorkers int, executor TaskExecutor) (*Orchestrator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	jobQ := newJobQueue(120)
	retryQ := newJobQueue(120)

	pool, err := ants.NewPool(maxWorkers,
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(1000),
		ants.WithExpiryDuration(5*time.Minute),
	)
	if err != nil {
		klog.Errorf("ants pool initialization failed: %v", err)
		return nil, err
	}

	return &Orchestrator{
		jobQueue:            jobQ,
		retryQueue:          retryQ,
		retryTicker:         time.NewTicker(500 * time.Millisecond),
		pool:                pool,
		activeCancellations: make(map[uint]context.CancelFunc),
		executor:            executor,
		ctx:                 ctx,
		cancel:              cancel,
	}, nil
}

// -----------------------------
// 启动
// -----------------------------
func (o *Orchestrator) Start() {
	go o.dispatchLoop()
	go o.processRetryQueue()
}

// -----------------------------
// 停止
// -----------------------------
func (o *Orchestrator) Stop() {
	o.stopOnce.Do(func() {
		klog.V(6).Infof("Orchestrator stopping...")

		// 1. 停止接收新任务，关闭队列
		o.cancel()
		o.jobQueue.Close()
		o.retryQueue.Close()

		// 2. 等待队列中待执行的任务全部分发完毕（原有逻辑保留）
		for {
			if o.jobQueue.Len() == 0 && o.retryQueue.Len() == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
			klog.V(6).Infof("Waiting for queues to empty: main=%d, retry=%d", o.jobQueue.Len(), o.retryQueue.Len())
		}

		// 3. 等待正在执行的长任务完成（核心适配 ants/v2）
		// 3.1 先打印当前运行中的任务数，便于排查
		runningTasks := o.pool.Running()
		if runningTasks > 0 {
			klog.V(6).Infof("Waiting for %d running tasks to complete (timeout: 35min)", runningTasks)
		}

		// 3.2 使用 ReleaseTimeout 等待35分钟（覆盖30分钟任务超时）
		// 该方法会阻塞，直到：1. 所有任务完成；2. 超时；3. 被中断
		timeout := 35 * time.Minute
		rErr := o.pool.ReleaseTimeout(timeout)

		// 3.3 打印等待结果日志
		if rErr == nil {
			klog.V(6).Infof("All running tasks completed before timeout")
		} else {
			klog.Warningf("Timeout after %v: some running tasks may be forced to stop", timeout)
		}

		klog.V(6).Infof("Orchestrator stopped completely")
	})
}

// -----------------------------
// 入队任务
// -----------------------------
func (o *Orchestrator) EnqueueJob(job *Job) error {
	select {
	case <-o.ctx.Done():
		return ErrOrchestratorStopped
	default:
	}

	if err := o.jobQueue.Enqueue(job); err != nil {
		if errors.Is(err, ErrQueueFull) {
			klog.Warningf("Job queue full: taskID=%d", job.TaskID)
		}
		return err
	}
	klog.V(6).Infof("Job enqueued: taskID=%d", job.TaskID)
	return nil
}

func (o *Orchestrator) EnqueueBatch(jobs []*Job) error {
	var failedJobs []*Job
	for _, job := range jobs {
		if err := o.EnqueueJob(job); err != nil {
			klog.Warningf("Batch enqueue failed for taskID=%d: %v", job.TaskID, err)
			failedJobs = append(failedJobs, job)
		}
	}
	if len(failedJobs) > 0 {
		return fmt.Errorf("failed to enqueue %d jobs (total %d)", len(failedJobs), len(jobs))
	}
	return nil
}

// -----------------------------
// 取消任务
// -----------------------------
func (o *Orchestrator) registerCancel(taskID uint, cancel context.CancelFunc) {
	o.cancelMutex.Lock()
	defer o.cancelMutex.Unlock()
	o.activeCancellations[taskID] = cancel
}

func (o *Orchestrator) unregisterCancel(taskID uint) {
	o.cancelMutex.Lock()
	defer o.cancelMutex.Unlock()
	delete(o.activeCancellations, taskID)
}

func (o *Orchestrator) CancelTask(taskID uint) bool {
	o.cancelMutex.Lock()
	cancel, ok := o.activeCancellations[taskID]
	o.cancelMutex.Unlock()
	if !ok {
		return false
	}

	klog.V(6).Infof("Cancelling task: taskID=%d", taskID)
	cancel()

	select {
	case <-time.After(5 * time.Second):
		klog.Warningf("Task cancel timeout: taskID=%d", taskID)
	case <-o.ctx.Done():
	}

	return true
}

// -----------------------------
// Dispatch Loop
// -----------------------------
func (o *Orchestrator) dispatchLoop() {
	for {
		select {
		case <-o.ctx.Done():
			return
		default:
			job, ok := o.jobQueue.Dequeue()
			if !ok {
				continue
			}
			// TODO 获取TASK，检查其RunAfter 的任务是否已完成，若未完成，则不入队
			// job.TaskID
			o.tryDispatch(job)
		}
	}
}

// -----------------------------
// Retry Queue Loop
// -----------------------------
func (o *Orchestrator) processRetryQueue() {
	defer o.retryTicker.Stop()
	// 增加协程级Panic防护，避免协程退出
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("Retry queue loop panic recovered: %v", r)
		}
	}()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-o.retryTicker.C:
			for range 10 {
				job, ok := o.retryQueue.Dequeue()
				if !ok {
					break
				}
				// 单个任务Panic不影响整个循环
				func() {
					defer func() {
						if r := recover(); r != nil {
							klog.Errorf("Retry dispatch panic: taskID=%d, err=%v",
								job.TaskID, r)
						}
					}()
					o.tryDispatch(job)
				}()
			}
		}
	}
}

// -----------------------------
// Try Dispatch
// -----------------------------
// tryDispatch
// 说明：尝试分发任务到协程池执行；当仓库锁被占用或池提交失败时，按重试上限与计数进行重试入队
// 参数：job 待执行的任务
// 行为：当达到重试上限时直接放弃，并打印中文日志
// tryDispatch 精简为只负责分发，不操作 RetryCount
func (o *Orchestrator) tryDispatch(job *Job) {

	if job.MaxRetries <= 0 || job.RetryCount >= job.MaxRetries {
		klog.Warningf("任务重试已达上限，放弃入队: taskID=%d, retry=%d/%d", job.TaskID, job.RetryCount, job.MaxRetries)
		return
	}
	if err := o.pool.Submit(func() {
		o.executeJob(job)
	}); err == nil {
		return
	} else {
		klog.Errorf("提交任务到协程池失败: taskID=%d, err=%v", job.TaskID, err)
	}

	if job.MaxRetries <= 0 || job.RetryCount >= job.MaxRetries {
		klog.Warningf("任务重试已达上限，放弃入队: taskID=%d, retry=%d/%d", job.TaskID, job.RetryCount, job.MaxRetries)
		return
	}
	job.RetryCount++
	if err := o.retryQueue.Enqueue(job); err != nil {
		klog.Errorf("任务重试入队失败: taskID=%d, err=%v", job.TaskID, err)
	}
}

// executeJob 统一控制重试
func (o *Orchestrator) executeJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("Task panic recovered: taskID=%d, err=%v", job.TaskID, r)
			o.unregisterCancel(job.TaskID)
		}
	}()

	timeout := job.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(o.ctx, timeout)
	defer cancel()
	runCtx, manualCancel := context.WithCancel(ctx)
	defer manualCancel()

	o.registerCancel(job.TaskID, manualCancel)
	defer o.unregisterCancel(job.TaskID)

	for i := job.RetryCount; i < job.MaxRetries; i++ {
		job.RetryCount = i // 每次尝试前更新 RetryCount

		err := o.executor.ExecuteTask(runCtx, job.TaskID)
		if err == nil {
			klog.V(6).Infof("Task completed: taskID=%d", job.TaskID)
			return
		}

		backoff := time.Second << i
		if backoff > 20*time.Minute {
			backoff = 20 * time.Minute
		}

		klog.Warningf("任务重试失败: taskID=%d, retry=%d/%d, err=%v, backoff=%v",
			job.TaskID, i+1, job.MaxRetries, err, backoff)

		select {
		case <-runCtx.Done():
			klog.Warningf("任务被取消或超时: taskID=%d", job.TaskID)
			return
		case <-time.After(backoff):
		}
	}

	klog.Errorf("任务执行失败且超过重试上限: taskID=%d", job.TaskID)
}

// -----------------------------
// Queue Status
// -----------------------------
type QueueStatus struct {
	QueueLength   int `json:"queue_length"`
	ActiveWorkers int `json:"active_workers"`
	ActiveRepos   int `json:"active_repos"`
}

func (o *Orchestrator) GetQueueStatus() *QueueStatus {
	return &QueueStatus{
		QueueLength:   o.jobQueue.Len(),
		ActiveWorkers: o.pool.Running(),
	}
}

// -----------------------------
// JobQueue (Ring Buffer) + Reject New
// -----------------------------
type jobQueue struct {
	maxSize int
	items   []*Job
	mutex   sync.Mutex
	cond    *sync.Cond
	closed  bool
}

func newJobQueue(maxSize int) *jobQueue {
	q := &jobQueue{
		maxSize: maxSize,
		items:   make([]*Job, 0, maxSize),
	}
	q.cond = sync.NewCond(&q.mutex)
	return q
}

func (q *jobQueue) Enqueue(job *Job) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.closed {
		return ErrOrchestratorStopped
	}
	if q.maxSize > 0 && len(q.items) >= q.maxSize {
		return ErrQueueFull // Reject New
	}
	q.items = append(q.items, job)
	q.cond.Signal()
	return nil
}

func (q *jobQueue) Dequeue() (*Job, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return nil, false
	}
	job := q.items[0]
	q.items[0] = nil
	q.items = q.items[1:]
	return job, true
}

func (q *jobQueue) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return len(q.items)
}

func (q *jobQueue) Close() {
	q.mutex.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mutex.Unlock()
}

// -------------------- Global Orchestrator --------------------
var (
	globalOrchestrator *Orchestrator
	orchestratorOnce   sync.Once
)

func InitGlobalOrchestrator(maxWorkers int, executor TaskExecutor) error {
	var initErr error
	orchestratorOnce.Do(func() {
		orch, err := NewOrchestrator(maxWorkers, executor)
		if err != nil {
			initErr = err
			return
		}
		globalOrchestrator = orch
		globalOrchestrator.Start()
		klog.V(6).Infof("Global orchestrator initialized: maxWorkers=%d", maxWorkers)
	})
	return initErr
}

func GetGlobalOrchestrator() *Orchestrator {
	return globalOrchestrator
}

func ShutdownGlobalOrchestrator() {
	if globalOrchestrator != nil {
		globalOrchestrator.Stop()
		klog.V(6).Infof("Global orchestrator shutdown")
	}
}
