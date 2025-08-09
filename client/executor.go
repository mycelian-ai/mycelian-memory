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

// Note: all clients include an executor by default; async methods require it.
