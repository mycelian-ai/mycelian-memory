package client

// add_entry_sdk_test.go exercises the SDK's local behaviour (FIFO order and
// back-pressure mapping) without talking to a live backend. It stubs out the
// ShardExecutor and uses httptest.Server to return HTTP 201.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/synapse/synapse-mcp-server/internal/shardqueue"
)

// Recording stub for FIFO order + count assertions.
type stubExec struct {
	mu    sync.Mutex
	keys  []string
	count int32
}

func (s *stubExec) Submit(ctx context.Context, key string, j shardqueue.Job) error {
	atomic.AddInt32(&s.count, 1)
	s.mu.Lock()
	s.keys = append(s.keys, key)
	s.mu.Unlock()
	if j != nil {
		_ = j.Run(ctx)
	}
	return nil
}
func (s *stubExec) Stop() {}

// Executor that always signals queue saturation.
type fullExec struct{}

func (fullExec) Submit(context.Context, string, shardqueue.Job) error { return shardqueue.ErrQueueFull }
func (fullExec) Stop()                                                {}

func TestAddEntry_SDKFIFOAndBackPressure(t *testing.T) {
	t.Parallel()

	// fake backend returning 201
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"entryId":"e1"}`))
	}))
	defer srv.Close()

	// ----- FIFO & Ack -----
	stub := &stubExec{}
	c := MustNew(srv.URL)
	c.overrideExecutor(stub)

	if _, err := c.AddEntry(context.Background(), "u1", "memX", AddEntryRequest{RawEntry: "one"}); err != nil {
		t.Fatalf("enqueue error: %v", err)
	}
	if _, err := c.AddEntry(context.Background(), "u1", "memX", AddEntryRequest{RawEntry: "two"}); err != nil {
		t.Fatalf("second enqueue error: %v", err)
	}

	if got := atomic.LoadInt32(&stub.count); got != 2 {
		t.Fatalf("expected 2 submits, got %d", got)
	}
	stub.mu.Lock()
	keysCopy := append([]string(nil), stub.keys...)
	stub.mu.Unlock()
	if keysCopy[0] != "memX" || keysCopy[1] != "memX" {
		t.Fatalf("FIFO violated, got %v", keysCopy)
	}

	// ----- back-pressure mapping -----
	bpClient := MustNew(srv.URL)
	bpClient.overrideExecutor(fullExec{})
	if _, err := bpClient.AddEntry(context.Background(), "u2", "memZ", AddEntryRequest{RawEntry: "x"}); !errors.Is(err, ErrBackPressure) {
		t.Fatalf("expected ErrBackPressure, got %v", err)
	}
}
