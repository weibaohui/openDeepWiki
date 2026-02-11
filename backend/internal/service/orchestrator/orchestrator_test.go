package orchestrator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeExecutor struct {
	err   error
	calls int32
}

func (f *fakeExecutor) ExecuteTask(ctx context.Context, taskID uint) error {
	atomic.AddInt32(&f.calls, 1)
	return f.err
}

func TestTryDispatchRepoLockedMaxRetries(t *testing.T) {
	executor := &fakeExecutor{}
	o, _ := NewOrchestrator(1, executor)
	o.retryTicker.Stop()
	defer o.pool.Release()

	job := &Job{
		TaskID:     1,
		RetryCount: 1,
		MaxRetries: 1,
		Timeout:    10 * time.Millisecond,
	}

	o.tryDispatch(job)

	if got := o.retryQueue.Len(); got != 0 {
		t.Fatalf("retry queue should be empty, got %d", got)
	}
	if atomic.LoadInt32(&executor.calls) != 0 {
		t.Fatalf("executor should not be called, got %d", executor.calls)
	}
	if job.RetryCount != 1 {
		t.Fatalf("retry count should remain 1, got %d", job.RetryCount)
	}
}

func TestTryDispatchRepoLockedEnqueueRetry(t *testing.T) {
	executor := &fakeExecutor{}
	o, _ := NewOrchestrator(1, executor)
	o.retryTicker.Stop()
	defer o.pool.Release()

	job := &Job{
		TaskID:     2,
		RetryCount: 0,
		MaxRetries: 1,
		Timeout:    10 * time.Millisecond,
	}

	o.tryDispatch(job)

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&executor.calls) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := o.retryQueue.Len(); got != 0 {
		t.Fatalf("retry queue should be empty, got %d", got)
	}
	if atomic.LoadInt32(&executor.calls) != 1 {
		t.Fatalf("executor should be called once, got %d", executor.calls)
	}
}

func TestExecuteJobStopsOnTimeout(t *testing.T) {
	executor := &fakeExecutor{err: context.DeadlineExceeded}
	o, _ := NewOrchestrator(1, executor)
	o.retryTicker.Stop()
	defer o.pool.Release()

	job := &Job{
		TaskID:     3,
		RetryCount: 0,
		MaxRetries: 3,
		Timeout:    50 * time.Millisecond,
	}

	start := time.Now()
	o.executeJob(job)
	elapsed := time.Since(start)

	if atomic.LoadInt32(&executor.calls) != 1 {
		t.Fatalf("executor should be called once, got %d", executor.calls)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("executeJob took too long: %v", elapsed)
	}
}
