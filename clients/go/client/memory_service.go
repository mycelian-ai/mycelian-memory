package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// listMemoriesResponse matches the backend response schema for list memories endpoints.
type listMemoriesResponse struct {
	Memories []Memory `json:"memories"`
	Count    int      `json:"count"`
}

// ListMemories returns all memories within a vault identified by UUID.
// Endpoint: GET /api/users/{userId}/vaults/{vaultId}/memories
func (c *Client) ListMemories(ctx context.Context, userID, vaultID string) ([]Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", c.baseURL, userID, vaultID)
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
		return nil, fmt.Errorf("list memories: status %d", resp.StatusCode)
	}

	var lr listMemoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Memories, nil
}

// ListMemoriesByVaultTitle lists all memories within a vault identified by its title.
// Endpoint: GET /api/users/{userId}/vaults/{vaultTitle}/memories
func (c *Client) ListMemoriesByVaultTitle(ctx context.Context, userID, vaultTitle string) ([]Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", c.baseURL, userID, vaultTitle)
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
		return nil, fmt.Errorf("list memories by title: status %d", resp.StatusCode)
	}

	var lr listMemoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Memories, nil
}

// GetMemoryByTitle retrieves a memory by vault title + memory title.
// Endpoint: GET /api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}
func (c *Client) GetMemoryByTitle(ctx context.Context, userID, vaultTitle, memoryTitle string) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", c.baseURL, userID, vaultTitle, memoryTitle)
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
		return nil, fmt.Errorf("get memory by title: status %d", resp.StatusCode)
	}

	var m Memory
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetVaultByTitle fetches a vault by its title.
// Endpoint: GET /api/users/{userId}/vaults/{vaultTitle}
func (c *Client) GetVaultByTitle(ctx context.Context, userID, vaultTitle string) (*Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s", c.baseURL, userID, vaultTitle)
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
		return nil, fmt.Errorf("get vault by title: status %d", resp.StatusCode)
	}

	var v Vault
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateMemoryInVault creates a memory under a specific vault.
// Endpoint: POST /api/users/{userId}/vaults/{vaultId}/memories
func (c *Client) CreateMemoryInVault(ctx context.Context, userID, vaultID string, req CreateMemoryRequest) (*Memory, error) {
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
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", c.baseURL, userID, vaultID)
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

// GetMemoryInVault fetches a memory by vault ID + memory ID.
// Endpoint: GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}
func (c *Client) GetMemoryInVault(ctx context.Context, userID, vaultID, memID string) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", c.baseURL, userID, vaultID, memID)
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
