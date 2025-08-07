package shardqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type jobFunc func(context.Context) error

func (f jobFunc) Run(ctx context.Context) error { return f(ctx) }

func TestShardExecutor_Retry(t *testing.T) {
	cfg := Config{Shards: 1, QueueSize: 10, MaxAttempts: 3, BaseBackoff: 10 * time.Millisecond}
	ex := NewShardExecutor(cfg)
	defer ex.Stop()

	var attempts int32
	job := jobFunc(func(ctx context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return context.DeadlineExceeded // arbitrary error
		}
		return nil
	})

	if err := ex.Submit(context.Background(), "k1", job); err != nil {
		t.Fatalf("submit: %v", err)
	}
	// wait for executor to drain
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
