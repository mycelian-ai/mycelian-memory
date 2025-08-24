package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/client/internal/api"
	"github.com/mycelian/mycelian-memory/client/internal/errors"
	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
	promptsinternal "github.com/mycelian/mycelian-memory/client/prompts"
	"github.com/mycelian/mycelian-memory/pkg/devauth"
	"github.com/rs/zerolog/log"
)

// Constants
const defaultUserAgent = "mycelian-memory-go-client"

const (
	defaultExecutorShards    = 4
	defaultExecutorQueueSize = 1000
)

// Errors are defined in errors.go

// See options.go for functional options

// Executor abstraction is in executor.go

// --------------------------------------------------------------------
// Client core
// --------------------------------------------------------------------

// Client is a lightweight, context-aware SDK for the Mycelian Memory Service.
// Responsibilities:
//   - Own an HTTP client and add the Authorization header on every request
//   - Own an async shard executor to preserve FIFO ordering per memory
//   - Expose thin, type-safe methods that forward to the internal API layer
//   - Provide client-side utilities like AwaitConsistency and embedded prompts

type Client struct {
	baseURL string
	http    *http.Client
	exec    executor
	apiKey  string // API key for actor authentication (must be explicitly configured)

	closedOnce uint32 // ensures Close is idempotent
}

// Verify Client implements io.Closer
var _ io.Closer = (*Client)(nil)

// New constructs a Client with the specified baseURL and apiKey.
// It returns an error for invalid inputs or option failures.
// Additional options can be provided via functional arguments.
func New(baseURL, apiKey string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}

	// Normalize baseURL to avoid trailing-slash issues when composing URLs
	baseURL = strings.TrimRight(baseURL, "/")

	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: http.DefaultTransport, // Initialize transport early
		},
	}

	// Auto-enable debug via env variable without changing code.
	if debugLoggingRequested() {
		opts = append(opts, WithDebugLogging(true))
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.exec == nil {
		c.exec = newDefaultExecutor()
	}

	// Wrap HTTP transport to automatically add Authorization header
	c.wrapTransportWithAPIKey()

	return c, nil
}

// NewWithDevMode constructs a Client for development mode using the shared dev API key.
// This only works when the server is running in development mode with MockAuthorizer.
// Convenience constructor for local development; not for production use.
func NewWithDevMode(baseURL string, opts ...Option) (*Client, error) {
	// Use the shared dev API key that MockAuthorizer recognizes
	return New(baseURL, devauth.APIKey, opts...)
}

// wrapTransportWithAPIKey wraps the HTTP client's transport so every request
// carries the Authorization header. This is the single authoritative place
// that sets the header for the SDK.
func (c *Client) wrapTransportWithAPIKey() {
	// Transport is guaranteed to be non-nil after constructor initialization
	c.http.Transport = &apiKeyTransport{
		base:   c.http.Transport,
		apiKey: c.apiKey,
	}
}

// apiKeyTransport wraps an http.RoundTripper to automatically add the
// Authorization header. A minimal default User-Agent is also added when absent
// to aid observability during debugging.
type apiKeyTransport struct {
	base   http.RoundTripper
	apiKey string
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	cloned := req.Clone(req.Context())
	// Set Authorization in one authoritative place
	cloned.Header.Set("Authorization", "Bearer "+t.apiKey)
	// Add a default User-Agent only if caller didn't set one
	if cloned.Header.Get("User-Agent") == "" {
		cloned.Header.Set("User-Agent", defaultUserAgent)
	}
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

// AwaitConsistency blocks until all previously submitted jobs for memoryID
// have been executed by the internal executor. It delegates to the executor's
// Barrier so the client does not manipulate jobs directly.
func (c *Client) AwaitConsistency(ctx context.Context, memoryID string) error {
	return c.exec.Barrier(ctx, memoryID)
}

// newDefaultExecutor constructs the shardqueue executor with sane defaults.
func newDefaultExecutor() *shardqueue.ShardExecutor {
	cfg := shardqueue.Config{
		Shards:    defaultExecutorShards,
		QueueSize: defaultExecutorQueueSize,
		// CRITICAL: Enhanced error handler with classification awareness
		ErrorHandler: func(err error) {
			// Check if this is a classified error
			if classifiedErr, ok := err.(*errors.ClassifiedError); ok {
				// Log with error classification context
				if classifiedErr.Category == errors.Irrecoverable {
					log.Error().
						Str("category", "IRRECOVERABLE").
						Int("status_code", classifiedErr.StatusCode).
						Str("response_body", classifiedErr.Body).
						Stack().Err(classifiedErr.Underlying).
						Msg("IRRECOVERABLE ERROR - failed immediately, no retries")

					// Explicit stderr logging for immediate visibility
					fmt.Fprintf(os.Stderr, "🚨 IRRECOVERABLE ERROR [HTTP %d]: %s\n",
						classifiedErr.StatusCode, classifiedErr.Body)
				} else {
					log.Error().
						Str("category", "RECOVERABLE").
						Int("status_code", classifiedErr.StatusCode).
						Stack().Err(classifiedErr.Underlying).
						Msg("RECOVERABLE ERROR - max retries exceeded")

					fmt.Fprintf(os.Stderr, "⚠️ RECOVERABLE ERROR [HTTP %d] - max retries exceeded: %v\n",
						classifiedErr.StatusCode, classifiedErr.Underlying)
				}
			} else {
				// Unclassified error - log with stack trace
				log.Error().Stack().Err(err).Msg("UNCLASSIFIED ASYNC JOB ERROR")
				fmt.Fprintf(os.Stderr, "🔥 UNCLASSIFIED ERROR: %v\n", err)
			}
		},
	}
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

// PutContext stores the plain-text context document via the sharded executor.
func (c *Client) PutContext(ctx context.Context, vaultID, memID string, doc string) (*EnqueueAck, error) {
	return api.PutContext(ctx, c.exec, c.http, c.baseURL, vaultID, memID, doc)
}

// GetLatestContext fetches the latest context document as plain text.
func (c *Client) GetLatestContext(ctx context.Context, vaultID, memID string) (string, error) {
	return api.GetLatestContext(ctx, c.http, c.baseURL, vaultID, memID)
}

// DeleteContext removes a context snapshot by ID synchronously via HTTP.
// It first awaits consistency to ensure all pending writes complete, then performs the deletion.
func (c *Client) DeleteContext(ctx context.Context, vaultID, memID, contextID string) error {
	return api.DeleteContext(ctx, c.exec, c.http, c.baseURL, vaultID, memID, contextID)
}

// --------------------------------------------------------------------
// Prompts operations - embedded (sync-only, no network)
// --------------------------------------------------------------------

// LoadDefaultPrompts returns the embedded default prompts for the given memory
// type (e.g., "chat", "code"). No network calls are made.
func (c *Client) LoadDefaultPrompts(ctx context.Context, memoryType string) (*DefaultPromptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return promptsinternal.LoadDefaultPrompts(memoryType)
}
