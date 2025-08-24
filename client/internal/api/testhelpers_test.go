package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
)

// errRT is an http.RoundTripper that always returns an error (simulates network failure).
type errRT struct{}

func (e *errRT) RoundTrip(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("boom") }

// failingExec implements types.Executor and always fails Submit.
type failingExec struct{}

func (f *failingExec) Submit(ctx context.Context, shard string, job shardqueue.Job) error {
	return fmt.Errorf("submit failed")
}
