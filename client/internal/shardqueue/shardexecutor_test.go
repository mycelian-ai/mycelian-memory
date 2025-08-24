package shardqueue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type noopJob struct{}

func (n noopJob) Run(ctx context.Context) error { return nil }

func TestShardExecutor_SubmitAndStop(t *testing.T) {
	t.Parallel()
	exec := NewShardExecutor(Config{})
	defer exec.Stop()

	if err := exec.Submit(context.Background(), "k1", noopJob{}); err != nil {
		t.Fatalf("submit error: %v", err)
	}
}

func TestShardExecutor_QueueFull(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	cfg.QueueSize = 1
	cfg.Shards = 1
	cfg.EnqueueTimeout = 10 * time.Millisecond
	exec := NewShardExecutor(cfg)
	defer exec.Stop()

	// Block the worker by submitting a job with a context that never completes until we cancel
	blockCtx, cancel := context.WithCancel(context.Background())
	var started int32
	_ = exec.Submit(context.Background(), "same", JobFunc(func(ctx context.Context) error {
		atomic.StoreInt32(&started, 1)
		<-blockCtx.Done()
		return nil
	}))

	// Wait until worker starts
	for atomic.LoadInt32(&started) == 0 {
		time.Sleep(time.Millisecond)
	}

	// Fill the buffer
	_ = exec.Submit(context.Background(), "same", noopJob{})
	if err := exec.Submit(context.Background(), "same", noopJob{}); err == nil {
		t.Fatal("expected queue full error")
	}
	cancel()
}

// ---------- helpers ----------

type testJob struct{ run func(context.Context) error }

func (t testJob) Run(ctx context.Context) error { return t.run(ctx) }

// reuse noopJob from above

// ---------- unit tests ----------

// FIFO ordering for a single key.
func TestShardExecutor_FIFOOrdering(t *testing.T) {
	p := NewShardExecutor(Config{Shards: 4, QueueSize: 10})
	defer p.Stop()

	var (
		mu    sync.Mutex
		order []int
		wg    sync.WaitGroup
	)
	wg.Add(5)
	for i := 0; i < 5; i++ {
		v := i
		if err := p.Submit(context.Background(), "mem1", testJob{run: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, v)
			mu.Unlock()
			wg.Done()
			return nil
		}}); err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for jobs")
	}

	for i, v := range order {
		if i != v {
			t.Fatalf("expected FIFO order, got %v", order)
		}
	}
}

// Jobs for different keys run in parallel (no head‑of‑line blocking).
func TestShardExecutor_ParallelDifferentKeys(t *testing.T) {
	p := NewShardExecutor(Config{Shards: 4, QueueSize: 10})
	defer p.Stop()

	start := make(chan struct{})
	done := make(chan struct{})

	_ = p.Submit(context.Background(), "A", testJob{run: func(context.Context) error {
		<-start
		close(done)
		return nil
	}})
	_ = p.Submit(context.Background(), "B", testJob{run: func(context.Context) error {
		close(start)
		<-done
		return nil
	}})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("jobs blocked each other; expected parallelism")
	}
}

// No overlap for the same key (serial execution guarantee).
func TestShardExecutor_SerialExecutionSameKey(t *testing.T) {
	const N = 200
	p := NewShardExecutor(Config{Shards: 4, QueueSize: N})
	defer p.Stop()

	var (
		inFlight        int32
		overlapDetected int32
		wg              sync.WaitGroup
	)
	wg.Add(N)

	for i := 0; i < N; i++ {
		_ = p.Submit(context.Background(), "X", testJob{run: func(context.Context) error {
			if atomic.AddInt32(&inFlight, 1) > 1 {
				atomic.StoreInt32(&overlapDetected, 1)
			}
			time.Sleep(100 * time.Microsecond)
			atomic.AddInt32(&inFlight, -1)
			wg.Done()
			return nil
		}})
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("serial execution test timed out")
	}

	if atomic.LoadInt32(&overlapDetected) == 1 {
		t.Fatal("detected overlapping execution for same key")
	}
}

// Submit after Stop should fail with ErrExecutorClosed.
func TestShardExecutor_SubmitAfterStop(t *testing.T) {
	p := NewShardExecutor(Config{Shards: 2, QueueSize: 2})
	p.Stop()

	err := p.Submit(context.Background(), "Z", noopJob{})
	if !errors.Is(err, ErrExecutorClosed) {
		t.Fatalf("expected ErrExecutorClosed, got %v", err)
	}
}

// Stop racing with many concurrent Submit calls should never panic or deadlock.
func TestShardExecutor_StopSubmit_RaceFree(t *testing.T) {
	p := NewShardExecutor(Config{Shards: 4, QueueSize: 32})

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.Submit(context.Background(), "k", noopJob{})
		}()
	}

	go p.Stop()
	wg.Wait()
}

// Two goroutines submit concurrently after Stop; both should see ErrExecutorClosed.
func TestConcurrentSubmitAfterStop(t *testing.T) {
	p := NewShardExecutor(Config{Shards: 2, QueueSize: 4})

	// Ensure some work has started so workers are alive.
	_ = p.Submit(context.Background(), "warm", noopJob{})

	// Kick off Stop and then two concurrent Submit calls.
	p.Stop()

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			errs <- p.Submit(context.Background(), "key", noopJob{})
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if !errors.Is(err, ErrExecutorClosed) {
			t.Fatalf("expected ErrExecutorClosed, got %v", err)
		}
	}
}
