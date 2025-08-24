package types

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
)

// ------------------------------
// Shared Interfaces
// ------------------------------

// Executor interface for dependency injection (used by async operations)
type Executor interface {
	Submit(context.Context, string, shardqueue.Job) error
}

// HTTPClient interface for dependency injection
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ------------------------------
// Shared Errors
// ------------------------------

// ErrNotFound is returned when context snapshot is not found
var ErrNotFound = fmt.Errorf("context snapshot not found")
