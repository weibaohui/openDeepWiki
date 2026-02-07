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
	TaskID       uint
	RepositoryID uint
	EnqueuedAt   time.Time
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

	repoConcurrency map[uint]bool
	repoMutex       sync.Mutex

	executor TaskExecutor

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	activeCancellations map[string]context.CancelFunc
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

// -----------------------------
// 工具函数
// -----------------------------
func getTaskKey(taskID, repoID uint) string {
	return fmt.Sprintf("%d:%d", taskID, repoID)
}

func NewTaskJob(taskID, repositoryID uint) *Job {
	return &Job{
		TaskID:       taskID,
		RepositoryID: repositoryID,
		EnqueuedAt:   time.Now(),
	}
}

// -----------------------------
// 构造函数
// -----------------------------
func NewOrchestrator(maxWorkers int, executor TaskExecutor) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	jobQ := newJobQueue(120)
	retryQ := newJobQueue(120)

	pool, err := ants.NewPool(maxWorkers,
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(1000),
		ants.WithExpiryDuration(5*time.Minute),
	)
	if err != nil {
		klog.Fatalf("ants pool initialization failed: %v", err)
	}

	return &Orchestrator{
		jobQueue:            jobQ,
		retryQueue:          retryQ,
		retryTicker:         time.NewTicker(500 * time.Millisecond),
		pool:                pool,
		repoConcurrency:     make(map[uint]bool),
		activeCancellations: make(map[string]context.CancelFunc),
		executor:            executor,
		ctx:                 ctx,
		cancel:              cancel,
	}
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

		o.cancel()
		o.jobQueue.Close()
		o.retryQueue.Close()

		for {
			if o.jobQueue.Len() == 0 && o.retryQueue.Len() == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
			klog.V(6).Infof("Waiting for queues to empty: main=%d, retry=%d", o.jobQueue.Len(), o.retryQueue.Len())
		}

		o.pool.Release()

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
			klog.Warningf("Job queue full: taskID=%d, repoID=%d", job.TaskID, job.RepositoryID)
		}
		return err
	}
	klog.V(6).Infof("Job enqueued: taskID=%d, repoID=%d", job.TaskID, job.RepositoryID)
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
func (o *Orchestrator) registerCancel(taskID, repoID uint, cancel context.CancelFunc) {
	o.cancelMutex.Lock()
	defer o.cancelMutex.Unlock()
	o.activeCancellations[getTaskKey(taskID, repoID)] = cancel
}

func (o *Orchestrator) unregisterCancel(taskID, repoID uint) {
	o.cancelMutex.Lock()
	defer o.cancelMutex.Unlock()
	delete(o.activeCancellations, getTaskKey(taskID, repoID))
}

func (o *Orchestrator) CancelTask(taskID, repoID uint) bool {
	o.cancelMutex.Lock()
	cancel, ok := o.activeCancellations[getTaskKey(taskID, repoID)]
	o.cancelMutex.Unlock()
	if !ok {
		return false
	}

	klog.V(6).Infof("Cancelling task: taskID=%d, repoID=%d", taskID, repoID)
	cancel()

	select {
	case <-time.After(5 * time.Second):
		klog.Warningf("Task cancel timeout: taskID=%d, repoID=%d", taskID, repoID)
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
				time.Sleep(50 * time.Millisecond)
				continue
			}
			o.tryDispatch(job)
		}
	}
}

// -----------------------------
// Retry Queue Loop
// -----------------------------
func (o *Orchestrator) processRetryQueue() {
	defer o.retryTicker.Stop()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-o.retryTicker.C:
			for i := 0; i < 10; i++ {
				job, ok := o.retryQueue.Dequeue()
				if !ok {
					break
				}
				o.tryDispatch(job)
			}
		}
	}
}

// -----------------------------
// Try Dispatch
// -----------------------------
func (o *Orchestrator) tryDispatch(job *Job) {
	if !o.acquireRepoLock(job.RepositoryID) {
		_ = o.retryQueue.Enqueue(job)
		return
	}

	lockReleased := false
	defer func() {
		if !lockReleased {
			o.releaseRepoLock(job.RepositoryID)
		}
	}()

	err := o.pool.Submit(func() {
		lockReleased = true
		defer o.releaseRepoLock(job.RepositoryID)
		o.executeJob(job)
	})
	if err != nil {
		klog.Errorf("Failed to submit job to pool: taskID=%d, repoID=%d, err=%v", job.TaskID, job.RepositoryID, err)
		_ = o.retryQueue.Enqueue(job)
	}
}

// -----------------------------
// Execute Job
// -----------------------------
func (o *Orchestrator) executeJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("Task panic recovered: taskID=%d, repoID=%d, err=%v", job.TaskID, job.RepositoryID, r)
			o.unregisterCancel(job.TaskID, job.RepositoryID)
		}
	}()

	ctx, cancel := context.WithTimeout(o.ctx, 10*time.Minute)
	defer cancel()
	runCtx, manualCancel := context.WithCancel(ctx)
	defer manualCancel()

	o.registerCancel(job.TaskID, job.RepositoryID, manualCancel)
	defer o.unregisterCancel(job.TaskID, job.RepositoryID)

	if err := o.executor.ExecuteTask(runCtx, job.TaskID); err != nil {
		klog.Errorf("Task failed: taskID=%d, repoID=%d, err=%v", job.TaskID, job.RepositoryID, err)
	} else {
		klog.V(6).Infof("Task completed: taskID=%d, repoID=%d", job.TaskID, job.RepositoryID)
	}
}

// -----------------------------
// Repo Lock
// -----------------------------
func (o *Orchestrator) acquireRepoLock(repoID uint) bool {
	o.repoMutex.Lock()
	defer o.repoMutex.Unlock()
	if o.repoConcurrency[repoID] {
		return false
	}
	o.repoConcurrency[repoID] = true
	return true
}

func (o *Orchestrator) releaseRepoLock(repoID uint) {
	o.repoMutex.Lock()
	defer o.repoMutex.Unlock()
	delete(o.repoConcurrency, repoID)
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
	o.repoMutex.Lock()
	defer o.repoMutex.Unlock()
	return &QueueStatus{
		QueueLength:   o.jobQueue.Len(),
		ActiveWorkers: o.pool.Running(),
		ActiveRepos:   len(o.repoConcurrency),
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

func InitGlobalOrchestrator(maxWorkers int, executor TaskExecutor) {
	orchestratorOnce.Do(func() {
		globalOrchestrator = NewOrchestrator(maxWorkers, executor)
		globalOrchestrator.Start()
		klog.V(6).Infof("Global orchestrator initialized: maxWorkers=%d", maxWorkers)
	})
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
