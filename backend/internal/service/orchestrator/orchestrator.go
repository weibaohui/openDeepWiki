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

// -------------------- Errors --------------------
var (
	// ErrRepoQueueFull occurs when the repo queue is full.
	ErrRepoQueueFull = errors.New("repo queue is full (reject new)")
)

// -------------------- Job --------------------
type Job struct {
	TaskID       uint
	RepositoryID uint
	EnqueuedAt   time.Time
}

// NewTaskJob creates a new Job
func NewTaskJob(taskID, repositoryID uint) *Job {
	return &Job{
		TaskID:       taskID,
		RepositoryID: repositoryID,
		EnqueuedAt:   time.Now(),
	}
}

// -------------------- Task Executor --------------------
type TaskExecutor interface {
	ExecuteTask(ctx context.Context, taskID uint) error
}

// -------------------- Orchestrator --------------------
type Orchestrator struct {
	jobQueue *jobQueue

	// ants worker pool (execution only)
	pool *ants.Pool

	// repo concurrency control
	repoConcurrency map[uint]bool
	repoMutex       sync.Mutex

	executor TaskExecutor

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	activeCancellations map[uint]context.CancelFunc
	cancelMutex         sync.Mutex
}

// NewOrchestrator creates a new Orchestrator
func NewOrchestrator(maxWorkers int, executor TaskExecutor) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	pool, err := ants.NewPool(maxWorkers, ants.WithNonblocking(true))
	if err != nil {
		panic(err)
	}

	return &Orchestrator{
		jobQueue:            newJobQueue(120),
		pool:                pool,
		repoConcurrency:     make(map[uint]bool),
		activeCancellations: make(map[uint]context.CancelFunc),
		executor:            executor,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start begins the orchestrator dispatch loop
func (o *Orchestrator) Start() {
	go o.dispatchLoop()
}

// Stop stops orchestrator and releases resources
func (o *Orchestrator) Stop() {
	o.stopOnce.Do(func() {
		o.cancel()
		o.jobQueue.Close()
		o.pool.Release()
		klog.V(6).Infof("Orchestrator stopped")
	})
}

// EnqueueJob submits a job to the queue
func (o *Orchestrator) EnqueueJob(job *Job) error {
	select {
	case <-o.ctx.Done():
		return fmt.Errorf("orchestrator is stopped")
	default:
	}

	if err := o.jobQueue.Enqueue(job); err != nil {
		if err.Error() == "task queue is full" {
			klog.Warningf("Job queue full: taskID=%d, repoID=%d", job.TaskID, job.RepositoryID)
		}
		return err
	}

	klog.V(6).Infof("Job enqueued: taskID=%d, repoID=%d", job.TaskID, job.RepositoryID)
	return nil
}

// EnqueueBatch submits multiple jobs
func (o *Orchestrator) EnqueueBatch(jobs []*Job) error {
	for _, job := range jobs {
		if err := o.EnqueueJob(job); err != nil {
			return err
		}
	}
	return nil
}

// CancelTask cancels a running task
func (o *Orchestrator) CancelTask(taskID uint) bool {
	o.cancelMutex.Lock()
	defer o.cancelMutex.Unlock()

	if cancel, ok := o.activeCancellations[taskID]; ok {
		klog.V(6).Infof("Cancelling running task: taskID=%d", taskID)
		cancel()
		return true
	}
	return false
}

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

// GetQueueStatus returns current queue metrics
func (o *Orchestrator) GetQueueStatus() *QueueStatus {
	o.repoMutex.Lock()
	defer o.repoMutex.Unlock()
	return &QueueStatus{
		QueueLength:   o.jobQueue.Len(),
		ActiveRepos:   len(o.repoConcurrency),
		ActiveWorkers: o.pool.Cap(),
	}
}

// -------------------- QueueStatus --------------------
type QueueStatus struct {
	QueueLength   int `json:"queue_length"`
	ActiveRepos   int `json:"active_repos"`
	ActiveWorkers int `json:"active_workers"`
}

// -------------------- Dispatch Loop --------------------
func (o *Orchestrator) dispatchLoop() {
	for {
		select {
		case <-o.ctx.Done():
			return
		default:
		}

		job, ok := o.jobQueue.Dequeue()
		if !ok {
			return
		}

		o.tryDispatch(job)
	}
}

func (o *Orchestrator) tryDispatch(job *Job) {
	if !o.acquireRepoLock(job.RepositoryID) {
		time.Sleep(2 * time.Second)
		_ = o.EnqueueJob(job)
		return
	}

	err := o.pool.Submit(func() {
		defer o.releaseRepoLock(job.RepositoryID)
		o.executeJob(job)
	})

	if err != nil {
		// pool full -> re-enqueue and release semaphore
		o.releaseRepoLock(job.RepositoryID)
		_ = o.EnqueueJob(job)
	}
}

func (o *Orchestrator) executeJob(job *Job) {
	ctx, cancel := context.WithTimeout(o.ctx, 10*time.Minute)
	defer cancel()

	runCtx, manualCancel := context.WithCancel(ctx)
	defer manualCancel()

	o.registerCancel(job.TaskID, manualCancel)
	defer o.unregisterCancel(job.TaskID)

	if err := o.executor.ExecuteTask(runCtx, job.TaskID); err != nil {
		klog.Errorf("task failed: taskID=%d err=%v", job.TaskID, err)
	} else {
		klog.V(6).Infof("task completed: taskID=%d repoID=%d", job.TaskID, job.RepositoryID)
	}
}

// -------------------- Repo Semaphore --------------------
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

// -------------------- Job Queue --------------------
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
		return errors.New("orchestrator is stopped")
	}
	if q.maxSize > 0 && len(q.items) >= q.maxSize {
		return ErrRepoQueueFull
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
