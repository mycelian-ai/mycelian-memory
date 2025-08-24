package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// Use shared types and validation from types package

// CreateUser registers a new user.
func CreateUser(ctx context.Context, httpClient *http.Client, baseURL string, req types.CreateUserRequest) (*types.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v0/users", baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create user: status %d", resp.StatusCode)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUser retrieves a user by ID.
func GetUser(ctx context.Context, httpClient *http.Client, baseURL, userID string) (*types.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Client-side validation removed; server is the authority
	url := fmt.Sprintf("%s/v0/users/%s", baseURL, userID)
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
		return nil, fmt.Errorf("get user: status %d", resp.StatusCode)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteUser removes a user by ID.
func DeleteUser(ctx context.Context, httpClient *http.Client, baseURL, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Client-side validation removed; server is the authority
	url := fmt.Sprintf("%s/v0/users/%s", baseURL, userID)
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
		return fmt.Errorf("delete user: status %d", resp.StatusCode)
	}
	return nil
}
