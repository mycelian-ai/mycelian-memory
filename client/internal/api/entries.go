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

// AddEntry submits a new entry to a memory via the sharded executor.
// This ensures FIFO ordering per memory and provides offline resilience.
// CRITICAL: This MUST preserve the async executor pattern!
func AddEntry(ctx context.Context, exec types.Executor, httpClient *http.Client, baseURL, vaultID, memID string, req types.AddEntryRequest) (*types.EnqueueAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Create job that makes the actual HTTP request
	addJob := job.New(func(jobCtx context.Context) error {
		body, err := json.Marshal(req)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s/entries", baseURL, vaultID, memID)
		httpReq, err := http.NewRequestWithContext(jobCtx, http.MethodPost, url, bytes.NewBuffer(body))
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
			return fmt.Errorf("add entry: status %d", resp.StatusCode)
		}
		return nil
	})

	// Submit job to executor for FIFO ordering per memory
	if err := exec.Submit(ctx, memID, addJob); err != nil {
		return nil, err
	}

	// Return acknowledgment that job was enqueued
	return &types.EnqueueAck{MemoryID: memID, Status: "enqueued"}, nil
}

// ListEntries retrieves entries within a memory using the full prefix (synchronous).
func ListEntries(ctx context.Context, httpClient *http.Client, baseURL, vaultID, memID string, params map[string]string) (*types.ListEntriesResponse, error) {
	if err := ctx.Err(); err != nil {
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
	url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s/entries%s", baseURL, vaultID, memID, query)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list entries: status %d", resp.StatusCode)
	}
	var lr types.ListEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}

// GetEntry retrieves a single entry by entryId within a memory (synchronous).
func GetEntry(ctx context.Context, httpClient *http.Client, baseURL, vaultID, memID, entryID string) (*types.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s/entries/%s", baseURL, vaultID, memID, entryID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get entry: status %d", resp.StatusCode)
	}
	var e types.Entry
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return nil, err
	}
	return &e, nil
}

// DeleteEntry removes an entry by ID from a memory synchronously.
// It first awaits consistency to ensure all pending writes complete, then performs the HTTP DELETE.
func DeleteEntry(ctx context.Context, exec types.Executor, httpClient *http.Client, baseURL, vaultID, memID, entryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Ensure all pending writes for this memory complete before deletion
	if err := awaitConsistency(ctx, exec, memID); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s/entries/%s", baseURL, vaultID, memID, entryID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete entry: status %d", resp.StatusCode)
	}
	return nil
}

// awaitConsistency blocks until all previously submitted jobs for the given memoryID
// have been executed by the internal executor. This ensures FIFO ordering is preserved.
func awaitConsistency(ctx context.Context, exec types.Executor, memoryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan struct{})
	job := job.New(func(context.Context) error {
		close(done)
		return nil
	})
	if err := exec.Submit(ctx, memoryID, job); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
