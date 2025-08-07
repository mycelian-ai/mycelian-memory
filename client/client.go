package client

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/client/internal/api"
	"github.com/mycelian/mycelian-memory/client/internal/job"
	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
	promptsinternal "github.com/mycelian/mycelian-memory/client/prompts"
)

// Errors moved to errors.go

// --------------------------------------------------------------------
// (Functional options moved to options.go)
// --------------------------------------------------------------------

// Executor abstraction lives in executor.go

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
	if debugLoggingRequested() {
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

// --------------------------------------------------------------------
// Memory operations - delegated to internal/api
// --------------------------------------------------------------------

// CreateMemory creates a new memory in the given vault.
func (c *Client) CreateMemory(ctx context.Context, userID, vaultID string, req CreateMemoryRequest) (*Memory, error) {
	return api.CreateMemory(ctx, c.http, c.baseURL, userID, vaultID, req)
}

// ListMemories retrieves memories within a vault.
func (c *Client) ListMemories(ctx context.Context, userID, vaultID string) ([]Memory, error) {
	return api.ListMemories(ctx, c.http, c.baseURL, userID, vaultID)
}

// GetMemory retrieves a specific memory.
func (c *Client) GetMemory(ctx context.Context, userID, vaultID, memoryID string) (*Memory, error) {
	return api.GetMemory(ctx, c.http, c.baseURL, userID, vaultID, memoryID)
}

// DeleteMemory deletes a specific memory.
func (c *Client) DeleteMemory(ctx context.Context, userID, vaultID, memoryID string) error {
	return api.DeleteMemory(ctx, c.http, c.baseURL, userID, vaultID, memoryID)
}

// --------------------------------------------------------------------
// Vault operations - delegated to internal/api
// --------------------------------------------------------------------

// CreateVault creates a new vault for the specified user.
func (c *Client) CreateVault(ctx context.Context, userID string, req CreateVaultRequest) (*Vault, error) {
	return api.CreateVault(ctx, c.http, c.baseURL, userID, req)
}

// ListVaults returns all vaults for a user.
func (c *Client) ListVaults(ctx context.Context, userID string) ([]Vault, error) {
	return api.ListVaults(ctx, c.http, c.baseURL, userID)
}

// GetVault retrieves a vault by ID.
func (c *Client) GetVault(ctx context.Context, userID, vaultID string) (*Vault, error) {
	return api.GetVault(ctx, c.http, c.baseURL, userID, vaultID)
}

// DeleteVault deletes the vault. Backend returns 204 No Content on success.
func (c *Client) DeleteVault(ctx context.Context, userID, vaultID string) error {
	return api.DeleteVault(ctx, c.http, c.baseURL, userID, vaultID)
}

// GetVaultByTitle fetches a vault by its title.
func (c *Client) GetVaultByTitle(ctx context.Context, userID, vaultTitle string) (*Vault, error) {
	return api.GetVaultByTitle(ctx, c.http, c.baseURL, userID, vaultTitle)
}

// --------------------------------------------------------------------
// User operations - delegated to internal/api
// --------------------------------------------------------------------

// CreateUser registers a new user.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	return api.CreateUser(ctx, c.http, c.baseURL, req)
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	return api.GetUser(ctx, c.http, c.baseURL, userID)
}

// DeleteUser removes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	return api.DeleteUser(ctx, c.http, c.baseURL, userID)
}

// --------------------------------------------------------------------
// Search operations - delegated to internal/api
// --------------------------------------------------------------------

// Search runs a search query against the backend.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	return api.Search(ctx, c.http, c.baseURL, req)
}

// --------------------------------------------------------------------
// Entry operations - delegated to internal/api (CRITICAL: mixed sync/async)
// --------------------------------------------------------------------

// AddEntry submits a new entry to a memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func (c *Client) AddEntry(ctx context.Context, userID, vaultID, memID string, req AddEntryRequest) (*EnqueueAck, error) {
	// CRITICAL: Pass the executor for async operation
	return api.AddEntry(ctx, c.exec, c.http, c.baseURL, userID, vaultID, memID, req)
}

// ListEntries retrieves entries within a memory using the full prefix (synchronous).
func (c *Client) ListEntries(ctx context.Context, userID, vaultID, memID string, params map[string]string) (*ListEntriesResponse, error) {
	return api.ListEntries(ctx, c.http, c.baseURL, userID, vaultID, memID, params)
}

// DeleteEntry removes an entry by ID from a memory via the sharded executor (async).
// This ensures FIFO ordering per memory and provides offline resilience.
func (c *Client) DeleteEntry(ctx context.Context, userID, vaultID, memID, entryID string) (*EnqueueAck, error) {
	// CRITICAL: Pass the executor for async operation
	return api.DeleteEntry(ctx, c.exec, c.http, c.baseURL, userID, vaultID, memID, entryID)
}

// --------------------------------------------------------------------
// Context operations - delegated to internal/api (CRITICAL: mixed sync/async)
// --------------------------------------------------------------------

// PutContext stores a snapshot for the memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func (c *Client) PutContext(ctx context.Context, userID, vaultID, memID string, payload PutContextRequest) (*EnqueueAck, error) {
	// CRITICAL: Pass the executor for async operation
	return api.PutContext(ctx, c.exec, c.http, c.baseURL, userID, vaultID, memID, payload)
}

// GetContext retrieves the most recent context snapshot for a memory (synchronous).
func (c *Client) GetContext(ctx context.Context, userID, vaultID, memID string) (*GetContextResponse, error) {
	return api.GetContext(ctx, c.http, c.baseURL, userID, vaultID, memID)
}

// --------------------------------------------------------------------
// Prompts operations - delegated to internal/api (sync-only)
// --------------------------------------------------------------------

// LoadDefaultPrompts returns the default prompts for the given memory type ("chat", "code", ...).
func (c *Client) LoadDefaultPrompts(ctx context.Context, memoryType string) (*promptsinternal.DefaultPromptResponse, error) {
	apiResponse, err := api.LoadDefaultPrompts(ctx, memoryType)
	if err != nil {
		return nil, err
	}

	// Convert API response to client response
	return &promptsinternal.DefaultPromptResponse{
		Version:             apiResponse.Version,
		ContextSummaryRules: apiResponse.ContextSummaryRules,
		Templates:           apiResponse.Templates,
	}, nil
}
