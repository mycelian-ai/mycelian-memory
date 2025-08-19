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
	"github.com/mycelian/mycelian-memory/devmode"
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
	apiKey  string // API key for actor authentication (must be explicitly configured)

	closedOnce uint32 // ensures Close is idempotent
}

// New constructs a Client with the specified baseURL and apiKey.
// Additional options can be provided via functional arguments.
func New(baseURL, apiKey string, opts ...Option) *Client {
	if baseURL == "" {
		panic("baseURL cannot be empty")
	}
	if apiKey == "" {
		panic("apiKey cannot be empty")
	}

	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
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

	// Wrap HTTP transport to automatically add Authorization header
	c.wrapTransportWithAPIKey()

	return c
}

// NewWithDevMode constructs a Client for development mode using the shared dev API key.
// This only works when the server is running in development mode with MockAuthorizer.
// Convenience constructor for local development.
func NewWithDevMode(baseURL string, opts ...Option) *Client {
	// Use the shared dev API key that MockAuthorizer recognizes
	return New(baseURL, devmode.APIKey, opts...)
}

// wrapTransportWithAPIKey wraps the HTTP client's transport to automatically
// add the Authorization header to all requests using the configured API key.
func (c *Client) wrapTransportWithAPIKey() {
	baseTransport := c.http.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	c.http.Transport = &apiKeyTransport{
		base:   baseTransport,
		apiKey: c.apiKey,
	}
}

// apiKeyTransport wraps an http.RoundTripper to automatically add Authorization header
type apiKeyTransport struct {
	base   http.RoundTripper
	apiKey string
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	cloned := req.Clone(req.Context())
	// Add the Authorization header with Bearer token
	cloned.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.base.RoundTrip(cloned)
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
func (c *Client) CreateMemory(ctx context.Context, vaultID string, req CreateMemoryRequest) (*Memory, error) {
	return api.CreateMemory(ctx, c.http, c.baseURL, vaultID, req)
}

// ListMemories retrieves memories within a vault.
func (c *Client) ListMemories(ctx context.Context, vaultID string) ([]Memory, error) {
	return api.ListMemories(ctx, c.http, c.baseURL, vaultID)
}

// GetMemory retrieves a specific memory.
func (c *Client) GetMemory(ctx context.Context, vaultID, memoryID string) (*Memory, error) {
	return api.GetMemory(ctx, c.http, c.baseURL, vaultID, memoryID)
}

// DeleteMemory deletes a specific memory.
func (c *Client) DeleteMemory(ctx context.Context, vaultID, memoryID string) error {
	return api.DeleteMemory(ctx, c.http, c.baseURL, vaultID, memoryID)
}

// --------------------------------------------------------------------
// Vault operations - delegated to internal/api
// --------------------------------------------------------------------

// CreateVault creates a new vault.
func (c *Client) CreateVault(ctx context.Context, req CreateVaultRequest) (*Vault, error) {
	return api.CreateVault(ctx, c.http, c.baseURL, req)
}

// ListVaults returns all vaults.
func (c *Client) ListVaults(ctx context.Context) ([]Vault, error) {
	return api.ListVaults(ctx, c.http, c.baseURL)
}

// GetVault retrieves a vault by ID.
func (c *Client) GetVault(ctx context.Context, vaultID string) (*Vault, error) {
	return api.GetVault(ctx, c.http, c.baseURL, vaultID)
}

// DeleteVault deletes the vault. Backend returns 204 No Content on success.
func (c *Client) DeleteVault(ctx context.Context, vaultID string) error {
	return api.DeleteVault(ctx, c.http, c.baseURL, vaultID)
}

// GetVaultByTitle fetches a vault by its title.
func (c *Client) GetVaultByTitle(ctx context.Context, vaultTitle string) (*Vault, error) {
	return api.GetVaultByTitle(ctx, c.http, c.baseURL, vaultTitle)
}

// --------------------------------------------------------------------
// User operations - REMOVED: user management is now external
// --------------------------------------------------------------------

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
func (c *Client) AddEntry(ctx context.Context, vaultID, memID string, req AddEntryRequest) (*EnqueueAck, error) {
	// CRITICAL: Pass the executor for async operation
	return api.AddEntry(ctx, c.exec, c.http, c.baseURL, vaultID, memID, req)
}

// ListEntries retrieves entries within a memory using the full prefix (synchronous).
func (c *Client) ListEntries(ctx context.Context, vaultID, memID string, params map[string]string) (*ListEntriesResponse, error) {
	return api.ListEntries(ctx, c.http, c.baseURL, vaultID, memID, params)
}

// GetEntry retrieves a single entry by entryId within a memory (synchronous).
func (c *Client) GetEntry(ctx context.Context, vaultID, memID, entryID string) (*Entry, error) {
	return api.GetEntry(ctx, c.http, c.baseURL, vaultID, memID, entryID)
}

// DeleteEntry removes an entry by ID from a memory synchronously via HTTP.
// It first awaits consistency to ensure all pending writes complete, then performs the deletion.
func (c *Client) DeleteEntry(ctx context.Context, vaultID, memID, entryID string) error {
	return api.DeleteEntry(ctx, c.exec, c.http, c.baseURL, vaultID, memID, entryID)
}

// --------------------------------------------------------------------
// Context operations - delegated to internal/api (CRITICAL: mixed sync/async)
// --------------------------------------------------------------------

// PutContext stores a snapshot for the memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func (c *Client) PutContext(ctx context.Context, vaultID, memID string, payload PutContextRequest) (*EnqueueAck, error) {
	// CRITICAL: Pass the executor for async operation
	return api.PutContext(ctx, c.exec, c.http, c.baseURL, vaultID, memID, payload)
}

// GetContext retrieves the most recent context snapshot for a memory (synchronous).
func (c *Client) GetContext(ctx context.Context, vaultID, memID string) (*GetContextResponse, error) {
	return api.GetContext(ctx, c.http, c.baseURL, vaultID, memID)
}

// DeleteContext removes a context snapshot by ID synchronously via HTTP.
// It first awaits consistency to ensure all pending writes complete, then performs the deletion.
func (c *Client) DeleteContext(ctx context.Context, vaultID, memID, contextID string) error {
	return api.DeleteContext(ctx, c.exec, c.http, c.baseURL, vaultID, memID, contextID)
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
