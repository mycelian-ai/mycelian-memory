package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Vault operations - all methods operate directly on Client

// listVaultsResponse mirrors the list endpoint response shape.
type listVaultsResponse struct {
	Vaults []Vault `json:"vaults"`
	Count  int     `json:"count"`
}

// CreateVault creates a new vault for the specified user.
func (c *Client) CreateVault(ctx context.Context, userID string, req CreateVaultRequest) (*Vault, error) {
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
	url := fmt.Sprintf("%s/api/users/%s/vaults", c.baseURL, userID)
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
		return nil, fmt.Errorf("create vault: status %d", resp.StatusCode)
	}

	var vault Vault
	if err := json.NewDecoder(resp.Body).Decode(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// ListVaults returns all vaults for a user.
func (c *Client) ListVaults(ctx context.Context, userID string) ([]Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults", c.baseURL, userID)
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
		return nil, fmt.Errorf("list vaults: status %d", resp.StatusCode)
	}

	var lr listVaultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Vaults, nil
}

// GetVault retrieves a vault by ID.
func (c *Client) GetVault(ctx context.Context, userID, vaultID string) (*Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s", c.baseURL, userID, vaultID)
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
		return nil, fmt.Errorf("get vault: status %d", resp.StatusCode)
	}

	var vault Vault
	if err := json.NewDecoder(resp.Body).Decode(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// DeleteVault deletes the vault. Backend returns 204 No Content on success.
func (c *Client) DeleteVault(ctx context.Context, userID, vaultID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateUserID(userID); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s", c.baseURL, userID, vaultID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete vault: status %d", resp.StatusCode)
	}
	return nil
}

// GetVaultByTitle fetches a vault by its title.
// Endpoint: GET /api/users/{userId}/vaults/{vaultTitle}
func (c *Client) GetVaultByTitle(ctx context.Context, userID, vaultTitle string) (*Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateUserID(userID); err != nil {
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
