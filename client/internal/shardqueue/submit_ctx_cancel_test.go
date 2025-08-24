package shardqueue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Submit should return ctx.Err when the caller context is canceled while waiting for a full queue.
func TestSubmit_ContextCanceledWhileWaiting(t *testing.T) {
	cfg := Config{Shards: 1, QueueSize: 1, EnqueueTimeout: time.Second}
	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	// Block the worker with a long job.
	blockCtx, cancelBlock := context.WithCancel(context.Background())
	var started int32
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		atomic.StoreInt32(&started, 1)
		<-blockCtx.Done()
		return nil
	})); err != nil {
		t.Fatalf("submit block job: %v", err)
	}

	// Wait for worker to start.
	for atomic.LoadInt32(&started) == 0 {
		time.Sleep(time.Millisecond)
	}

	// Fill the buffer with one more job so the next submit will block on send.
	_ = ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error { return nil }))

	// Now attempt to submit with an already-canceled context; since queue is full, ctx should win.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ex.Submit(ctx, "k", JobFunc(func(ctx context.Context) error { return nil }))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	cancelBlock() // unblock worker to let test exit quickly
}
