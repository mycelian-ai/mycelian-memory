package client

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/client/internal/job"
)

func TestAwaitConsistency(t *testing.T) {
	c := New("http://example.com")

	memID := "mem-123"
	var ranFirst int32

	// enqueue a dummy job then barrier
	if err := c.exec.Submit(context.Background(), memID, job.New(func(ctx context.Context) error {
		time.Sleep(30 * time.Millisecond)
		atomic.StoreInt32(&ranFirst, 1)
		return nil
	})); err != nil {
		t.Fatalf("submit: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	if err := c.AwaitConsistency(ctx, memID); err != nil {
		t.Fatalf("await consistency: %v", err)
	}
	elapsed := time.Since(start)

	if atomic.LoadInt32(&ranFirst) == 0 {
		t.Fatalf("barrier returned before previous job executed")
	}

	if elapsed < 25*time.Millisecond {
		t.Fatalf("awaitConsistency returned too quickly: %v", elapsed)
	}
}
