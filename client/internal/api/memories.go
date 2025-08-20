package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// Use shared validation from types package

// Use shared types from types package

// CreateMemory creates a new memory in the given vault using API key authentication.
func CreateMemory(ctx context.Context, httpClient *http.Client, baseURL, vaultID string, req types.CreateMemoryRequest) (*types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/vaults/%s/memories", baseURL, vaultID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Note: Authorization header will be added by transport layer

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create memory: status %d", resp.StatusCode)
	}

	var mem types.Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// ListMemories retrieves memories within a vault using API key authentication.
func ListMemories(ctx context.Context, httpClient *http.Client, baseURL, vaultID string) ([]types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	url := fmt.Sprintf("%s/v0/vaults/%s/memories", baseURL, vaultID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list memories: status %d", resp.StatusCode)
	}

	var lr types.ListMemoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Memories, nil
}

// GetMemory retrieves a specific memory using API key authentication.
func GetMemory(ctx context.Context, httpClient *http.Client, baseURL, vaultID, memoryID string) (*types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s", baseURL, vaultID, memoryID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get memory: status %d", resp.StatusCode)
	}

	var mem types.Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// DeleteMemory deletes a specific memory using API key authentication.
func DeleteMemory(ctx context.Context, httpClient *http.Client, baseURL, vaultID, memoryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Client-side validation removed; server is the authority
	url := fmt.Sprintf("%s/v0/vaults/%s/memories/%s", baseURL, vaultID, memoryID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete memory: status %d", resp.StatusCode)
	}
	return nil
}
