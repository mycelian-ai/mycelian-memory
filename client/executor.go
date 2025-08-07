package client

import (
	"context"

	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
)

// executor abstracts the internal async job runner used by async APIs.
type executor interface {
	Submit(context.Context, string, shardqueue.Job) error
	Stop()
}

// noOpExecutor disables async APIs; calling async methods will panic with a
// clear error to surface misuse in short‑lived sync contexts (e.g., CLIs).
type noOpExecutor struct{}

func (noOpExecutor) Submit(context.Context, string, shardqueue.Job) error {
	panic("attempted to use async operation (AddEntry/PutContext/DeleteEntry) on sync-only client")
}
func (noOpExecutor) Stop() {}
