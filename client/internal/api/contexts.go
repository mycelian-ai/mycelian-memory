package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/job"
	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// Use shared types, validation, and interfaces from types package

// Use the shared ErrNotFound from types to ensure equality works across boundaries.

// PutContext stores a snapshot for the memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func PutContext(ctx context.Context, exec types.Executor, httpClient *http.Client, baseURL, userID, vaultID, memID string, payload types.PutContextRequest) (*types.EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(memID, "memoryId"); err != nil {
		return nil, err
	}

	// Create job that makes the actual HTTP request
	putJob := job.New(func(jobCtx context.Context) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/v0/users/%s/vaults/%s/memories/%s/contexts", baseURL, userID, vaultID, memID)
		httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodPut, url, bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("put context: status %d", resp.StatusCode)
		}
		return nil
	})

	// Submit job to executor for FIFO ordering per memory
	if err := exec.Submit(ctx, memID, putJob); err != nil {
		return nil, err
	}

	// Return acknowledgment that job was enqueued
	return &types.EnqueueAck{MemoryID: memID, Status: "enqueued"}, nil
}

// GetContext retrieves the most recent context snapshot for a memory (synchronous).
func GetContext(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memID string) (*types.GetContextResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(memID, "memoryId"); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/users/%s/vaults/%s/memories/%s/contexts", baseURL, userID, vaultID, memID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		var res types.GetContextResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}
		return &res, nil
	case http.StatusNotFound:
		return nil, types.ErrNotFound
	default:
		return nil, fmt.Errorf("get context: status %d", resp.StatusCode)
	}
}

// DeleteContext removes a context snapshot by contextId synchronously.
// Server treats contexts as append-only snapshots; delete is hard and irreversible.
func DeleteContext(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memID, contextID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return err
	}
	if err := types.ValidateIDPresent(memID, "memoryId"); err != nil {
		return err
	}
	if err := types.ValidateIDPresent(contextID, "contextId"); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v0/users/%s/vaults/%s/memories/%s/contexts/%s", baseURL, userID, vaultID, memID, contextID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete context: status %d", resp.StatusCode)
	}
	return nil
}
