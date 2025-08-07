package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Memory operations - all methods operate directly on Client

// listMemoriesResponse mirrors the backend list shape.
type listMemoriesResponse struct {
	Memories []Memory `json:"memories"`
	Count    int      `json:"count"`
}

// CreateMemory creates a new memory in the given vault.
func (c *Client) CreateMemory(ctx context.Context, userID, vaultID string, req CreateMemoryRequest) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create memory: status %d", resp.StatusCode)
	}

	var mem Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// ListMemories retrieves memories within a vault.
func (c *Client) ListMemories(ctx context.Context, userID, vaultID string) ([]Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list memories: status %d", resp.StatusCode)
	}

	var lr listMemoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Memories, nil
}

// GetMemory retrieves a specific memory.
func (c *Client) GetMemory(ctx context.Context, userID, vaultID, memoryID string) (*Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", c.baseURL, userID, vaultID, memoryID)
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
		return nil, fmt.Errorf("get memory: status %d", resp.StatusCode)
	}

	var mem Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// DeleteMemory deletes a specific memory.
func (c *Client) DeleteMemory(ctx context.Context, userID, vaultID, memoryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateUserID(userID); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", c.baseURL, userID, vaultID, memoryID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete memory: status %d", resp.StatusCode)
	}
	return nil
}
