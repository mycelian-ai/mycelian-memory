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
func AddEntry(ctx context.Context, exec types.Executor, httpClient *http.Client, baseURL, userID, vaultID, memID string, req types.AddEntryRequest) (*types.EnqueueAck, error) {
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
	addJob := job.New(func(jobCtx context.Context) error {
		body, err := json.Marshal(req)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries", baseURL, userID, vaultID, memID)
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
func ListEntries(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memID string, params map[string]string) (*types.ListEntriesResponse, error) {
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
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries%s", baseURL, userID, vaultID, memID, query)
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
func GetEntry(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memID, entryID string) (*types.Entry, error) {
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
	if err := types.ValidateIDPresent(entryID, "entryId"); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries/%s", baseURL, userID, vaultID, memID, entryID)
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

// DeleteEntry removes an entry by ID from a memory via the sharded executor (async).
// This ensures FIFO ordering per memory and provides offline resilience.
func DeleteEntry(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memID, entryID string) error {
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
	if err := types.ValidateIDPresent(entryID, "entryId"); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries/%s", baseURL, userID, vaultID, memID, entryID)
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
