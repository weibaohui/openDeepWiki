package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Job 定义编排器中的任务
type Job struct {
	TaskID       uint
	RepositoryID uint
	EnqueuedAt   time.Time
	Priority     int // 优先级，数值越大优先级越高
}

// TaskExecutor 任务执行器接口
// 抽象任务执行逻辑，便于测试和扩展
// 通过接口定义，避免循环依赖
type TaskExecutor interface {
	ExecuteTask(ctx context.Context, taskID uint) error
}

// Orchestrator 任务编排器
// 负责管理任务队列、worker池和任务执行
type Orchestrator struct {
	// 任务队列
	jobQueue      chan *Job
	priorityQueue chan *Job // 高优先级队列

	// Worker池配置
	maxWorkers      int
	workers         []*worker
	workerWg        sync.WaitGroup

	// 并发控制
	repoConcurrency map[uint]bool      // 记录每个仓库是否有任务在执行
	repoMutex      sync.Mutex         // 保护 repoConcurrency

	// 执行器
	executor TaskExecutor

	// 生命周期管理
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once
}

// NewOrchestrator 创建新的任务编排器
// maxWorkers: 最大worker数量（建议2-4，避免打爆CPU/LLM配额）
// executor: 任务执行器
func NewOrchestrator(maxWorkers int, executor TaskExecutor) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Orchestrator{
		jobQueue:        make(chan *Job, 100),  // 普通任务队列，缓冲100
		priorityQueue:   make(chan *Job, 20),   // 高优先级队列，缓冲20
		maxWorkers:      maxWorkers,
		workers:         make([]*worker, 0, maxWorkers),
		repoConcurrency: make(map[uint]bool),
		executor:        executor,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start 启动编排器
func (o *Orchestrator) Start() {
	klog.V(6).Infof("任务编排器启动中: maxWorkers=%d", o.maxWorkers)

	// 创建并启动worker
	for i := 0; i < o.maxWorkers; i++ {
		w := &worker{
			id:           i,
			orchestrator: o,
		}
		o.workers = append(o.workers, w)
		o.workerWg.Add(1)

		go w.Run()
	}

	klog.V(6).Infof("任务编排器启动完成: workers=%d", len(o.workers))
}

// Stop 停止编排器
func (o *Orchestrator) Stop() {
	o.stopOnce.Do(func() {
		klog.V(6).Infof("任务编排器停止中...")

		// 取消上下文
		o.cancel()

		// 等待所有worker完成
		o.workerWg.Wait()

		// 关闭队列
		close(o.jobQueue)
		close(o.priorityQueue)

		klog.V(6).Infof("任务编排器已停止")
	})
}

// EnqueueJob 提交任务到队列
// 自动根据优先级选择队列
func (o *Orchestrator) EnqueueJob(job *Job) error {
	select {
	case <-o.ctx.Done():
		return fmt.Errorf("orchestrator is stopped")
	default:
	}

	// 根据优先级选择队列
	if job.Priority > 0 {
		// 高优先级任务
		select {
		case o.priorityQueue <- job:
			klog.V(6).Infof("任务已入队(高优先级): taskID=%d, repoID=%d, priority=%d",
				job.TaskID, job.RepositoryID, job.Priority)
			return nil
		case <-o.ctx.Done():
			return fmt.Errorf("orchestrator is stopped")
		default:
			// 高优先级队列满了，尝试普通队列
		}
	}

	// 普通任务
	select {
	case o.jobQueue <- job:
		klog.V(6).Infof("任务已入队: taskID=%d, repoID=%d, priority=%d",
			job.TaskID, job.RepositoryID, job.Priority)
		return nil
	case <-o.ctx.Done():
		return fmt.Errorf("orchestrator is stopped")
	}
}

// EnqueueBatch 批量提交任务到队列
// 用于 run-all 场景，按 sort_order 顺序入队
func (o *Orchestrator) EnqueueBatch(jobs []*Job) error {
	klog.V(6).Infof("批量提交任务到队列: count=%d", len(jobs))

	for _, job := range jobs {
		if err := o.EnqueueJob(job); err != nil {
			klog.Errorf("批量提交任务失败: taskID=%d, error=%v", job.TaskID, err)
			return err
		}
	}

	return nil
}

// GetQueueStatus 获取队列状态信息
func (o *Orchestrator) GetQueueStatus() *QueueStatus {
	o.repoMutex.Lock()
	defer o.repoMutex.Unlock()

	return &QueueStatus{
		QueueLength:     len(o.jobQueue),
		PriorityLength:  len(o.priorityQueue),
		ActiveWorkers:   len(o.workers),
		ActiveRepos:     len(o.repoConcurrency),
	}
}

// QueueStatus 队列状态
type QueueStatus struct {
	QueueLength     int `json:"queue_length"`     // 普通队列长度
	PriorityLength  int `json:"priority_length"`  // 高优先级队列长度
	ActiveWorkers   int `json:"active_workers"`   // 活跃worker数
	ActiveRepos     int `json:"active_repos"`     // 活跃仓库数
}

// worker 工作线程
type worker struct {
	id           int
	orchestrator *Orchestrator
}

// Run 运行worker
func (w *worker) Run() {
	defer w.orchestrator.workerWg.Done()

	klog.V(6).Infof("Worker启动: id=%d", w.id)

	for {
		select {
		case <-w.orchestrator.ctx.Done():
			klog.V(6).Infof("Worker停止: id=%d", w.id)
			return

		// 优先处理高优先级队列
		case job := <-w.orchestrator.priorityQueue:
			w.processJob(job)

		// 处理普通队列
		case job := <-w.orchestrator.jobQueue:
			w.processJob(job)
		}
	}
}

// processJob 处理单个任务
func (w *worker) processJob(job *Job) {
	klog.V(6).Infof("Worker开始处理任务: workerID=%d, taskID=%d, repoID=%d",
		w.id, job.TaskID, job.RepositoryID)

	// 检查仓库并发限制
	if !w.acquireRepoLock(job.RepositoryID) {
		klog.V(6).Infof("仓库已有任务在执行，重新入队: repoID=%d, taskID=%d",
			job.RepositoryID, job.TaskID)
		// 重新入队，等待下次调度
		_ = w.orchestrator.EnqueueJob(job)
		return
	}
	defer w.releaseRepoLock(job.RepositoryID)

	// 创建任务上下文
	ctx, cancel := context.WithTimeout(w.orchestrator.ctx, 10*time.Minute)
	defer cancel()

	// 执行任务
	err := w.orchestrator.executor.ExecuteTask(ctx, job.TaskID)

	if err != nil {
		klog.Errorf("任务执行失败: workerID=%d, taskID=%d, repoID=%d, error=%v",
			w.id, job.TaskID, job.RepositoryID, err)
	} else {
		klog.V(6).Infof("任务执行完成: workerID=%d, taskID=%d, repoID=%d",
			w.id, job.TaskID, job.RepositoryID)
	}
}

// acquireRepoLock 获取仓库并发锁
// 返回true表示获取成功，false表示仓库已有任务在执行
func (w *worker) acquireRepoLock(repoID uint) bool {
	w.orchestrator.repoMutex.Lock()
	defer w.orchestrator.repoMutex.Unlock()

	if w.orchestrator.repoConcurrency[repoID] {
		return false
	}

	w.orchestrator.repoConcurrency[repoID] = true
	return true
}

// releaseRepoLock 释放仓库并发锁
func (w *worker) releaseRepoLock(repoID uint) {
	w.orchestrator.repoMutex.Lock()
	defer w.orchestrator.repoMutex.Unlock()

	delete(w.orchestrator.repoConcurrency, repoID)
}

// NewTaskJob 创建任务Job
// 自动设置入队时间和优先级
func NewTaskJob(taskID, repositoryID uint, priority int) *Job {
	return &Job{
		TaskID:       taskID,
		RepositoryID: repositoryID,
		EnqueuedAt:   time.Now(),
		Priority:     priority,
	}
}

// 全局编排器实例
var (
	globalOrchestrator *Orchestrator
	orchestratorOnce  sync.Once
)

// InitGlobalOrchestrator 初始化全局编排器
// 应该在应用启动时调用
func InitGlobalOrchestrator(maxWorkers int, executor TaskExecutor) {
	orchestratorOnce.Do(func() {
		globalOrchestrator = NewOrchestrator(maxWorkers, executor)
		globalOrchestrator.Start()
		klog.V(6).Infof("全局任务编排器已初始化: maxWorkers=%d", maxWorkers)
	})
}

// GetGlobalOrchestrator 获取全局编排器
func GetGlobalOrchestrator() *Orchestrator {
	return globalOrchestrator
}

// ShutdownGlobalOrchestrator 关闭全局编排器
// 应该在应用关闭时调用
func ShutdownGlobalOrchestrator() {
	if globalOrchestrator != nil {
		globalOrchestrator.Stop()
		klog.V(6).Infof("全局任务编排器已关闭")
	}
}
