package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mycelian/mycelian-memory/clients/go/client/internal/job"
)

// Context operations - PutContext is ASYNC (uses executor), GetContext is sync

// Context snapshot types are now in types.go

// PutContext stores a snapshot for the memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func (c *Client) PutContext(ctx context.Context, userID, vaultID, memID string, payload PutContextRequest) (*PutContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}

	// Create job that makes the actual HTTP request
	putJob := job.New(func(jobCtx context.Context) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/contexts", c.baseURL, userID, vaultID, memID)
		httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodPut, url, bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("put context: status %d", resp.StatusCode)
		}
		return nil
	})

	// Submit job to executor for FIFO ordering per memory
	if err := c.exec.Submit(ctx, memID, putJob); err != nil {
		return nil, err
	}

	// Return acknowledgment that job was enqueued
	return &PutContextResponse{
		UserID:       userID,
		MemoryID:     memID,
		ContextID:    "enqueued", // Will be set properly when job executes
		CreationTime: time.Now(),
	}, nil
}

// GetContext retrieves the most recent context snapshot for a memory (synchronous).
func (c *Client) GetContext(ctx context.Context, userID, vaultID, memID string) (*GetContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
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
	defer resp.Body.Close()
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
