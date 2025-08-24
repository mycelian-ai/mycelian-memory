package shardqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// When a job's context is canceled before the worker starts it, the worker should skip Run and invoke the error handler with ctx.Err.
func TestWorker_SkipsRunForCanceledJob(t *testing.T) {
	var handlerCalls int32
	cfg := Config{Shards: 1, QueueSize: 2, MaxAttempts: 1}
	cfg.ErrorHandler = func(err error) { atomic.AddInt32(&handlerCalls, 1) }

	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	// First job blocks the worker.
	blockCtx, unblock := context.WithCancel(context.Background())
	started := make(chan struct{})
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		close(started)
		<-blockCtx.Done()
		return nil
	})); err != nil {
		t.Fatalf("submit blocking job: %v", err)
	}
	<-started

	// Second job is queued behind the blocking one but will have its context canceled before execution.
	ran := int32(0)
	jobCtx, cancelJob := context.WithCancel(context.Background())
	if err := ex.Submit(jobCtx, "k", JobFunc(func(ctx context.Context) error {
		atomic.StoreInt32(&ran, 1)
		return nil
	})); err != nil {
		t.Fatalf("submit second job: %v", err)
	}

	// Cancel before worker gets to the job so run is skipped.
	cancelJob()

	// Unblock worker to move on to the canceled job.
	unblock()

	// Give some time for processing.
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&ran) == 1 {
		t.Fatal("job Run should not have been called for canceled context")
	}
	if atomic.LoadInt32(&handlerCalls) == 0 {
		t.Fatal("expected error handler to be invoked for canceled job")
	}
}
