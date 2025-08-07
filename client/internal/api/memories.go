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

// CreateMemory creates a new memory in the given vault.
func CreateMemory(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID string, req types.CreateMemoryRequest) (*types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return nil, err
	}
	if err := types.ValidateTitle(req.Title, "title"); err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", baseURL, userID, vaultID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

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

// ListMemories retrieves memories within a vault.
func ListMemories(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID string) ([]types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories", baseURL, userID, vaultID)
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

// GetMemory retrieves a specific memory.
func GetMemory(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memoryID string) (*types.Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return nil, err
	}
	if err := types.ValidateIDPresent(memoryID, "memoryId"); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", baseURL, userID, vaultID, memoryID)
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

// DeleteMemory deletes a specific memory.
func DeleteMemory(ctx context.Context, httpClient *http.Client, baseURL, userID, vaultID, memoryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := types.ValidateUserID(userID); err != nil {
		return err
	}
	if err := types.ValidateIDPresent(vaultID, "vaultId"); err != nil {
		return err
	}
	if err := types.ValidateIDPresent(memoryID, "memoryId"); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s", baseURL, userID, vaultID, memoryID)
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
