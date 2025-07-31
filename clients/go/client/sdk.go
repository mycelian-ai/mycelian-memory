package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/synapse/synapse-mcp-server/internal/shardqueue"
)

//--------------------------------------------------------------------
// Public error & helper
//--------------------------------------------------------------------

var ErrBackPressure = errors.New("back‑pressure (queue full)")

func IsBackPressure(err error) bool { return errors.Is(err, ErrBackPressure) }

//--------------------------------------------------------------------
// Debug transport wrapper
//--------------------------------------------------------------------

// debugTransport wraps an http.RoundTripper to log requests and responses
type debugTransport struct {
	base http.RoundTripper
}

func (dt *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if debug logging is enabled via env var
	if os.Getenv("SYNAPSE_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		// Log request
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Debug().
				Str("method", req.Method).
				Str("url", req.URL.String()).
				Str("request_dump", string(reqDump)).
				Msg("HTTP request")
		}
	}

	// Make the actual request
	resp, err := dt.base.RoundTrip(req)
	if err != nil {
		if os.Getenv("SYNAPSE_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
			log.Error().
				Err(err).
				Str("method", req.Method).
				Str("url", req.URL.String()).
				Msg("HTTP request failed")
		}
		return nil, err
	}

	// Log response if debug enabled
	if os.Getenv("SYNAPSE_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		respDump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Debug().
				Str("method", req.Method).
				Str("url", req.URL.String()).
				Int("status_code", resp.StatusCode).
				Str("response_dump", string(respDump)).
				Msg("HTTP response")
		}
	}

	return resp, nil
}

//--------------------------------------------------------------------
// Functional‑options constructor
//--------------------------------------------------------------------

type Option func(*Client) error

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("nil http client")
		}
		c.http = hc
		return nil
	}
}

func WithDebugLogging(enabled bool) Option {
	return func(c *Client) error {
		if enabled {
			// Wrap the transport with debug logging
			transport := c.http.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			c.http.Transport = &debugTransport{base: transport}
		}
		return nil
	}
}

// WithoutExecutor disables the shardqueue executor for clients that only use
// synchronous operations (CreateUser, CreateMemory, ListEntries, etc).
// This avoids the overhead of creating worker goroutines and queues.
// WARNING: Calling async methods (AddEntry, PutContext) on a client created
// with this option will panic.
func WithoutExecutor() Option {
	return func(c *Client) error {
		c.exec = &noOpExecutor{}
		return nil
	}
}

// noOpExecutor is used for sync-only clients
type noOpExecutor struct{}

func (noOpExecutor) Submit(context.Context, string, shardqueue.Job) error {
	panic("attempted to use async operation (AddEntry/PutContext) on sync-only client")
}

func (noOpExecutor) Stop() {
	// No-op since there are no workers to stop
}

//--------------------------------------------------------------------
// Client
//--------------------------------------------------------------------

// executor abstracts shardqueue.ShardExecutor for tests.
type executor interface {
	Submit(context.Context, string, shardqueue.Job) error
	Stop()
}

type Client struct {
	baseURL string
	http    *http.Client

	exec executor

	closedOnce uint32
}

func New(base string, opts ...Option) *Client {
	c := &Client{
		baseURL: base,
		http:    &http.Client{Timeout: 30 * time.Second},
	}

	// Enable debug logging if environment variable is set
	if os.Getenv("SYNAPSE_DEBUG") == "true" || os.Getenv("DEBUG") == "true" {
		opts = append(opts, WithDebugLogging(true))
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			panic(err)
		}
	}
	if c.exec == nil {
		ex, err := newDefaultExecutor()
		if err != nil {
			panic(err)
		}
		c.exec = ex
	}
	return c
}

// Close is idempotent.  If the client owns the executor it drains it.
func (c *Client) Close() error {
	if !atomic.CompareAndSwapUint32(&c.closedOnce, 0, 1) {
		return nil
	}
	if c.exec != nil {
		c.exec.Stop()
	}
	return nil
}

//--------------------------------------------------------------------
// Write path – enqueue‑only acknowledgement
//--------------------------------------------------------------------

// EnqueueAck is returned by AddEntry to confirm the job was accepted.
type EnqueueAck struct {
	MemoryID string `json:"memoryId"`
	Status   string `json:"status"` // always "enqueued" for now
}

// AddEntry submits an entry to the per-key executor, preserving FIFO order. It
// waits for the HTTP write to complete so callers get the created Entry back.
// Use await_consistency + read on the backend side if you need visibility
// guarantees across replicas.
func (c *Client) AddEntry(ctx context.Context, userID, memID string, req AddEntryRequest) (*EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	job := jobFunc(func(jctx context.Context) error {
		// convert panics into errors so ErrorHandler records them once
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic: %v", r)
				entriesFailedTotal.WithLabelValues(shardLabel(memID)).Inc()
				log.Error().Err(err).Msg("entry write failed")
			}
		}()

		// Execute HTTP write but ignore result—errors propagate to executor
		_, err := c.doAddEntry(jctx, userID, memID, req)
		return err
	})

	if err := c.exec.Submit(ctx, memID, job); err != nil {
		log.Warn().Str("memory_id", memID).Err(err).Msg("enqueue failed")
		if errors.Is(err, shardqueue.ErrQueueFull) {
			return nil, ErrBackPressure
		}
		return nil, err
	}

	entriesEnqueuedTotal.WithLabelValues(shardLabel(memID)).Inc()
	return &EnqueueAck{MemoryID: memID, Status: "enqueued"}, nil
}

//--------------------------------------------------------------------
// Internal helpers
//--------------------------------------------------------------------

func (c *Client) doAddEntry(ctx context.Context, userID, memID string, reqPayload AddEntryRequest) (*Entry, error) {
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/memories/%s/entries", c.baseURL, userID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var e Entry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}

func newDefaultExecutor() (*shardqueue.ShardExecutor, error) {
	cfg, err := shardqueue.LoadConfig()
	if err != nil {
		return nil, err
	}

	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = func(err error) {
			if err == nil {
				return
			}
			// log and metric
			log.Error().Err(err).Msg("entry write failed")
			// shard unknown; use label "-1"
			entriesFailedTotal.WithLabelValues("-1").Inc()
		}
	}

	return shardqueue.NewShardExecutor(cfg), nil
}

//--------------------------------------------------------------------
// Read / Write helpers – other endpoints (unchanged from previous SDK)
//--------------------------------------------------------------------

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.UserID == "" {
		return nil, fmt.Errorf("userId is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create user: status %d", resp.StatusCode)
	}
	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUser fetches a user.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user: status %d", resp.StatusCode)
	}
	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateMemory creates a memory for a user.
func (c *Client) CreateMemory(ctx context.Context, userID string, req CreateMemoryRequest) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/memories", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create memory: status %d", resp.StatusCode)
	}
	var m Memory
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMemory retrieves a memory.
func (c *Client) GetMemory(ctx context.Context, userID, memID string) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/memories/%s", c.baseURL, userID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get memory: status %d", resp.StatusCode)
	}
	var m Memory
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListEntries lists entries for a memory.
func (c *Client) ListEntries(ctx context.Context, userID, memID string, params map[string]string) (*ListEntriesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	query := ""
	first := true
	for k, v := range params {
		if first {
			query += "?"
			first = false
		} else {
			query += "&"
		}
		query += fmt.Sprintf("%s=%s", k, v)
	}
	url := fmt.Sprintf("%s/api/users/%s/memories/%s/entries%s", c.baseURL, userID, memID, query)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list entries: status %d", resp.StatusCode)
	}
	var lr ListEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}

// DeleteMemory deletes a memory and all its entries. Expected status 204 No Content.
func (c *Client) DeleteMemory(ctx context.Context, userID, memID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := requireUserID(userID); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/users/%s/memories/%s", c.baseURL, userID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete memory: status %d", resp.StatusCode)
	}
	return nil
}

// DeleteUser deletes a user and all subordinate resources. Expected status 204 No Content.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := requireUserID(userID); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/users/%s", c.baseURL, userID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete user: status %d", resp.StatusCode)
	}
	return nil
}

// MustNew remains for compatibility but simply calls New.
func MustNew(base string, opts ...Option) *Client {
	return New(base, opts...)
}

// AwaitConsistency enqueues a no-op barrier job for the memory shard and
// blocks until that job is executed. Because ShardExecutor preserves FIFO
// order per key, this guarantees that all previously submitted jobs for the
// same memory have completed (successfully or with their error propagated via
// the executor ErrorHandler). It makes no assumption about replication lag on
// the backend; it merely waits for local write completion.
func (c *Client) AwaitConsistency(ctx context.Context, memID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	done := make(chan struct{})
	barrier := jobFunc(func(jctx context.Context) error {
		close(done)
		return nil
	})

	if err := c.exec.Submit(ctx, memID, barrier); err != nil {
		if errors.Is(err, shardqueue.ErrQueueFull) {
			return ErrBackPressure
		}
		return err
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

//--------------------------------------------------------------------
// Context document APIs (filesystem-backed)
//--------------------------------------------------------------------

// PutContext enqueues a job to write the provided content as the activeContext
// for the given memory. It returns immediately with an EnqueueAck.
func (c *Client) PutContext(ctx context.Context, userID, memID, content string) (*EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	// Wrap raw text under activeContext key to preserve previous prompt contract
	payload := PutContextRequest{Context: map[string]string{"activeContext": content}}

	job := jobFunc(func(jctx context.Context) error {
		_, err := c.putContextHTTP(jctx, userID, memID, payload)
		return err
	})

	if err := c.exec.Submit(ctx, memID, job); err != nil {
		if errors.Is(err, shardqueue.ErrQueueFull) {
			return nil, ErrBackPressure
		}
		return nil, err
	}
	return &EnqueueAck{MemoryID: memID, Status: "enqueued"}, nil
}
