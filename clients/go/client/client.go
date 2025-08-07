package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"os"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/clients/go/client/internal/job"
	"github.com/mycelian/mycelian-memory/clients/go/client/internal/shardqueue"
	"github.com/rs/zerolog/log"
)

// --------------------------------------------------------------------
// Public errors & helpers
// --------------------------------------------------------------------

// ErrBackPressure is returned when the client's internal shard queue is full.
var ErrBackPressure = errors.New("back-pressure (queue full)")

// IsBackPressure reports whether err is a back-pressure error.
func IsBackPressure(err error) bool { return errors.Is(err, ErrBackPressure) }

// --------------------------------------------------------------------
// debugTransport – optional HTTP round-trip logger
// --------------------------------------------------------------------

type debugTransport struct{ base http.RoundTripper }

func (dt *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Str("request_dump", string(reqDump)).Msg("HTTP request")
		}
	}

	resp, err := dt.base.RoundTrip(req)
	if err != nil {
		if os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
			log.Error().Err(err).Str("method", req.Method).Str("url", req.URL.String()).Msg("HTTP request failed")
		}
		return nil, err
	}

	if os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		respDump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Int("status_code", resp.StatusCode).Str("response_dump", string(respDump)).Msg("HTTP response")
		}
	}
	return resp, nil
}

// --------------------------------------------------------------------
// (Functional options moved to options.go)
// --------------------------------------------------------------------

// --------------------------------------------------------------------
// Executor abstraction (internal)
// --------------------------------------------------------------------

type executor interface {
	Submit(context.Context, string, shardqueue.Job) error
	Stop()
}

type noOpExecutor struct{}

func (noOpExecutor) Submit(context.Context, string, shardqueue.Job) error {
	panic("attempted to use async operation (AddEntry/PutContext) on sync-only client")
}
func (noOpExecutor) Stop() {}

// --------------------------------------------------------------------
// Client core
// --------------------------------------------------------------------

type Client struct {
	baseURL string
	http    *http.Client
	exec    executor

	closedOnce uint32 // ensures Close is idempotent
}

// New constructs a Client with optional functional arguments.
func New(base string, opts ...Option) *Client {
	c := &Client{
		baseURL: base,
		http:    &http.Client{Timeout: 30 * time.Second},
	}

	// Auto-enable debug via env variable without changing code.
	if os.Getenv("MYCELIAN_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		opts = append(opts, WithDebugLogging(true))
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			panic(err)
		}
	}
	if c.exec == nil {
		c.exec = newDefaultExecutor()
	}

	return c
}

// MustNew constructs a Client with panic-on-error semantics (for testing).
func MustNew(base string, opts ...Option) *Client {
	return New(base, opts...)
}

// Close stops the background executor (if any). Safe to call multiple times.
func (c *Client) Close() error {
	if !atomic.CompareAndSwapUint32(&c.closedOnce, 0, 1) {
		return nil
	}
	if c.exec != nil {
		c.exec.Stop()
	}
	return nil
}

// AwaitConsistency blocks until all previously submitted jobs for the given memoryID
// have been executed by the internal executor. It works by submitting a no-op job
// and waiting for it to run, thereby guaranteeing FIFO ordering has flushed.
func (c *Client) AwaitConsistency(ctx context.Context, memoryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan struct{})
	job := job.New(func(context.Context) error {
		close(done)
		return nil
	})
	if err := c.exec.Submit(ctx, memoryID, job); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// newDefaultExecutor constructs the shardqueue executor with sane defaults.
func newDefaultExecutor() *shardqueue.ShardExecutor {
	cfg := shardqueue.Config{Shards: 4, QueueSize: 1000}
	return shardqueue.NewShardExecutor(cfg)
}
