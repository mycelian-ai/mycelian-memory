package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/synapse/synapse-mcp-server/internal/shardqueue"
)

//--------------------------------------------------------------------
// Context API (HTTP – introduced in backend API v3)
//--------------------------------------------------------------------

// PutContextRequest wraps the context payload under the mandatory top-level
// "context" key as required by the backend.
// The payload can be any non-empty JSON object (nested maps/arrays allowed).
//
// Example:
//
//	req := client.PutContextRequest{Context: map[string]any{"agenda": "Review"}}
//
// Values are left as interface{} to let callers send arbitrary JSON and let
// the backend perform validation.
//
// NOTE: Keys must be strings; leaf values must be non-empty strings or nested
// JSON. The client does not enforce these rules, mirroring the backend's
// behaviour.
//
// Endpoint: PUT /api/users/{userId}/memories/{memoryId}/contexts
type PutContextRequest struct {
	Context any `json:"context"`
}

// PutContextResponse is returned by a successful 201 Created.
// Endpoint: PUT /api/users/{userId}/memories/{memoryId}/contexts
//
// The server writes an append-only row and returns identifiers plus the write
// timestamp.
//
// Example JSON body:
//  {
//    "userId": "user-123…",
//    "memoryId": "mem-456…",
//    "contextId": "ctx-789…",
//    "creationTime": "2025-01-01T01:00:00Z"
//  }
//
// Any additional fields returned by future versions are ignored by the client
// thanks to `json.Unmarshal` behaviour.
//
// Endpoint: 201 response.
//
// We use time.Time to parse RFC-3339 timestamps.
// The field tags match the backend spec exactly.
//
// NOTE: creationTime is always present; we parse it but do not validate its
// monotonicity.
//
// See docs/reference/api-documentation-v3.md for full details.
//
// The same struct is reused for GET responses with an added `Context` field.

type PutContextResponse struct {
	UserID       string    `json:"userId"`
	MemoryID     string    `json:"memoryId"`
	ContextID    string    `json:"contextId"`
	CreationTime time.Time `json:"creationTime"`
}

// GetContextResponse extends PutContextResponse with the actual context
// object.
// Endpoint: GET /api/users/{userId}/memories/{memoryId}/contexts
// Status: 200 OK
//
// When no snapshot exists the backend replies 404.
// Best practice: SDK returns (nil, ErrNotFound).

// ErrNotFound is returned when the backend responds 404.
// Re-exporting os.ErrNotExist would leak fs semantics; define our own sentinel.
var ErrNotFound = fmt.Errorf("context snapshot not found")

type GetContextResponse struct {
	PutContextResponse
	Context any `json:"context"`
}

//--------------------------------------------------------------------
// Client methods
//--------------------------------------------------------------------

// PutContext sends the context snapshot to the backend and returns the row
// metadata. It performs a synchronous HTTP call; callers may follow up with
// GetLatestContext for read-after-write verification if needed.
func (c *Client) putContextHTTP(ctx context.Context, userID, memID string, payload PutContextRequest) (*PutContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/%s/memories/%s/contexts", c.baseURL, userID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
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
		return nil, fmt.Errorf("put context: status %d", resp.StatusCode)
	}

	var res PutContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetLatestContext fetches the most recent context snapshot for the specified
// memory. Returns (nil, ErrNotFound) when the backend replies 404.
func (c *Client) GetLatestContext(ctx context.Context, userID, memID string) (*GetContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/%s/memories/%s/contexts", c.baseURL, userID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		var res GetContextResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}
		return &res, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("get context: status %d", resp.StatusCode)
	}
}

// putContextHTTPInVault is the vault-aware version of putContextHTTP.
func (c *Client) putContextHTTPScoped(ctx context.Context, userID, vaultID, memID string, payload PutContextRequest) (*PutContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/contexts", c.baseURL, userID, vaultID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
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
		return nil, fmt.Errorf("put context: status %d", resp.StatusCode)
	}

	var res PutContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetLatestMemoryContext fetches the latest snapshot using vaultId.
func (c *Client) GetLatestMemoryContext(ctx context.Context, userID, vaultID, memID string) (*GetContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/contexts", c.baseURL, userID, vaultID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		var res GetContextResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}
		return &res, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("get context: status %d", resp.StatusCode)
	}
}

// PutMemoryContext enqueues a write job for the provided content (activeContext) under the specified memory.
func (c *Client) PutMemoryContext(ctx context.Context, userID, vaultID, memID, content string) (*EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	payload := PutContextRequest{Context: map[string]string{"activeContext": content}}

	job := jobFunc(func(jctx context.Context) error {
		_, err := c.putContextHTTPScoped(jctx, userID, vaultID, memID, payload)
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
