package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/clients/go/client/internal/job"
)

// Entry operations - AddEntry is ASYNC (uses executor), others are sync

// AddEntry submits a new entry to a memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func (c *Client) AddEntry(ctx context.Context, userID, vaultID, memID string, req AddEntryRequest) (*EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}

	// Create job that makes the actual HTTP request
	addJob := job.New(func(jobCtx context.Context) error {
		body, err := json.Marshal(req)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries", c.baseURL, userID, vaultID, memID)
		httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodPost, url, bytes.NewBuffer(body))
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
			return fmt.Errorf("add entry: status %d", resp.StatusCode)
		}
		return nil
	})

	// Submit job to executor for FIFO ordering per memory
	if err := c.exec.Submit(ctx, memID, addJob); err != nil {
		return nil, err
	}

	// Return acknowledgment that job was enqueued
	return &EnqueueAck{MemoryID: memID, Status: "enqueued"}, nil
}

// ListEntries retrieves entries within a memory using the full prefix (synchronous).
func (c *Client) ListEntries(ctx context.Context, userID, vaultID, memID string, params map[string]string) (*ListEntriesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
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
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries%s", c.baseURL, userID, vaultID, memID, query)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list entries: status %d", resp.StatusCode)
	}
	var lr ListEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}

// DeleteEntry removes an entry by ID from a memory via the sharded executor (async).
// This ensures FIFO ordering per memory and provides offline resilience.
func (c *Client) DeleteEntry(ctx context.Context, userID, vaultID, memID, entryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateUserID(userID); err != nil {
		return err
	}

	// Create job that makes the actual HTTP request
	deleteJob := job.New(func(jobCtx context.Context) error {
		url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries/%s", c.baseURL, userID, vaultID, memID, entryID)
		httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodDelete, url, nil)
		if err != nil {
			return err
		}
		resp, err := c.http.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("delete entry: status %d", resp.StatusCode)
		}
		return nil
	})

	// Submit job to executor for FIFO ordering per memory
	return c.exec.Submit(ctx, memID, deleteJob)
}
