package shardqueue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Error handler is invoked exactly once when a job returns an error and MaxAttempts=1.
func TestErrorHandler_CalledOnce(t *testing.T) {
	var calls int32
	cfg := Config{Shards: 1, QueueSize: 8, MaxAttempts: 1}
	cfg.ErrorHandler = func(err error) { atomic.AddInt32(&calls, 1) }

	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	// First job errors to trigger handler.
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		return errors.New("boom")
	})); err != nil {
		t.Fatalf("submit error job: %v", err)
	}

	// Second job signals completion so we know processing drained.
	done := make(chan struct{})
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		close(done)
		return nil
	})); err != nil {
		t.Fatalf("submit follow-up job: %v", err)
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for follow-up job")
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("error handler calls = %d, want 1", got)
	}
}

// Panic inside ErrorHandler must be recovered and must not crash the worker; subsequent jobs still run.
func TestErrorHandler_PanicRecovered(t *testing.T) {
	cfg := Config{Shards: 1, QueueSize: 8, MaxAttempts: 1}
	cfg.ErrorHandler = func(err error) { panic("handler panic") }

	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	// Trigger the error handler.
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		return errors.New("boom")
	})); err != nil {
		t.Fatalf("submit error job: %v", err)
	}

	// This job should still run even though the handler panicked.
	ran := make(chan struct{})
	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		close(ran)
		return nil
	})); err != nil {
		t.Fatalf("submit follow-up job: %v", err)
	}

	select {
	case <-ran:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not continue after handler panic")
	}
}

// With a nil ErrorHandler, errors are ignored (no panic, no crash) and subsequent jobs run.
func TestErrorHandler_Nil_NoCrash(t *testing.T) {
	cfg := Config{Shards: 1, QueueSize: 4, MaxAttempts: 1}
	// ErrorHandler is nil

	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	if err := ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		return errors.New("ignored")
	})); err != nil {
		t.Fatalf("submit: %v", err)
	}

	done := make(chan struct{})
	_ = ex.Submit(context.Background(), "k", JobFunc(func(ctx context.Context) error {
		close(done)
		return nil
	}))

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for job after ignored error")
	}
}
