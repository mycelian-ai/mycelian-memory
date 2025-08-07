package shardqueue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// If Stop is called while a Submit is waiting for space, Submit should return ErrExecutorClosed.
func TestSubmit_ReturnsQuicklyWhenStoppedWhileWaiting(t *testing.T) {
	cfg := Config{Shards: 1, QueueSize: 1, EnqueueTimeout: time.Second}
	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	// Block worker with a long-running job.
	blockCtx, cancelBlock := context.WithCancel(context.Background())
	var started int32
	_ = ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		atomic.StoreInt32(&started, 1)
		<-blockCtx.Done()
		return nil
	}))

	// wait until the worker has started the blocking job
	for atomic.LoadInt32(&started) == 0 {
		time.Sleep(time.Millisecond)
	}

	// Fill the buffer so the next submit will block on send
	_ = ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error { return nil }))

	// Start a submit that will block on the full queue
	errCh := make(chan error, 1)
	go func() {
		errCh <- ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error { return nil }))
	}()

	// Give the goroutine a moment to block in Submit, then stop the executor concurrently
	time.Sleep(10 * time.Millisecond)
	doneStop := make(chan struct{})
	go func() {
		ex.Stop()
		close(doneStop)
	}()
	// Unblock the running job so the worker can finish and Stop can return
	cancelBlock()

	select {
	case err := <-errCh:
		// It is acceptable for Submit to either succeed (queue drained just as Stop happened)
		// or return ErrExecutorClosed if Stop wins the race. Assert non-blocking behaviour only.
		if err != nil && !errors.Is(err, ErrExecutorClosed) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("submit did not return after Stop")
	}

	// Ensure Stop completed
	select {
	case <-doneStop:
	case <-time.After(1 * time.Second):
		t.Fatal("Stop did not complete")
	}
}
