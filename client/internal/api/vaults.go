package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// Use shared validation and types from types package

// CreateVault creates a new vault using API key authentication.
func CreateVault(ctx context.Context, httpClient *http.Client, baseURL string, req types.CreateVaultRequest) (*types.Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/vaults", baseURL)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create vault: status %d", resp.StatusCode)
	}

	var vault types.Vault
	if err := json.NewDecoder(resp.Body).Decode(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// ListVaults returns all vaults using API key authentication.
func ListVaults(ctx context.Context, httpClient *http.Client, baseURL string) ([]types.Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/vaults", baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Note: Authorization header will be added by transport layer
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list vaults: status %d", resp.StatusCode)
	}

	var lr types.ListVaultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Vaults, nil
}

// GetVault retrieves a vault by ID using API key authentication.
func GetVault(ctx context.Context, httpClient *http.Client, baseURL, vaultID string) (*types.Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/vaults/%s", baseURL, vaultID)
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
		return nil, fmt.Errorf("get vault: status %d", resp.StatusCode)
	}

	var vault types.Vault
	if err := json.NewDecoder(resp.Body).Decode(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// DeleteVault deletes the vault using API key authentication. Backend returns 204 No Content on success.
func DeleteVault(ctx context.Context, httpClient *http.Client, baseURL, vaultID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/v0/vaults/%s", baseURL, vaultID)
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
		return fmt.Errorf("delete vault: status %d", resp.StatusCode)
	}
	return nil
}

// GetVaultByTitle fetches a vault by its title using API key authentication.
func GetVaultByTitle(ctx context.Context, httpClient *http.Client, baseURL, vaultTitle string) (*types.Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/vaults/%s", baseURL, vaultTitle)
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
		return nil, fmt.Errorf("get vault by title: status %d", resp.StatusCode)
	}

	var v types.Vault
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
