package api

import (
	"context"
	"sync"

	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
)

// mockExec is a test helper that records submitted shards and runs jobs inline.
type mockExec struct {
	mu    sync.Mutex
	n     int
	calls []string
}

func (m *mockExec) Submit(ctx context.Context, shard string, job shardqueue.Job) error {
	m.mu.Lock()
	m.n++
	m.calls = append(m.calls, shard)
	m.mu.Unlock()
	return job.Run(ctx)
}
