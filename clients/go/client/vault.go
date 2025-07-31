package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Vault represents a collection of memories belonging to a user.
// It maps 1-to-many with memories and is uniquely identified by a server-generated
// UUIDv4 (vaultId) while retaining a human-readable `title` provided by the user.
//
// Field names and JSON tags follow the backend REST specification exactly.
// See docs/reference/api-documentation.md → Vault API.
//
// All timestamps are RFC-3339 and parsed into time.Time.
//
// NOTE: The backend currently enforces title uniqueness per user and only allows
// ASCII lower-case letters, digits, and hyphens; the SDK purposely does not
// repeat validation logic but provides helper functions where appropriate.
type Vault struct {
	UserID       string    `json:"userId"`
	VaultID      string    `json:"vaultId"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// CreateVaultRequest is the payload for POST /api/users/{userId}/vaults
// Only Title is mandatory; Description is optional.
type CreateVaultRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

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
	if err := requireUserID(userID); err != nil {
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

	var v Vault
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVault retrieves a vault by ID.
func (c *Client) GetVault(ctx context.Context, userID, vaultID string) (*Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
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

	var v Vault
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVaults returns all vaults for a user.
func (c *Client) ListVaults(ctx context.Context, userID string) ([]Vault, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
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

// DeleteVault deletes the vault. The backend returns 204 No Content when the
// vault is empty and deletion succeeds.
func (c *Client) DeleteVault(ctx context.Context, userID, vaultID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := requireUserID(userID); err != nil {
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
