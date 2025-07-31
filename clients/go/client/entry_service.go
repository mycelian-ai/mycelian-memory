package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AddEntryInVault submits a new entry to a memory identified by vaultId + memoryId.
// Unlike the legacy AddEntry it requires vaultId for path construction.
func (c *Client) AddEntryInVault(ctx context.Context, userID, vaultID, memID string, req AddEntryRequest) (*EnqueueAck, error) {
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

	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries", c.baseURL, userID, vaultID, memID)
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
		return nil, fmt.Errorf("add entry: status %d", resp.StatusCode)
	}

	// Backend returns the created Entry; SDK mirrors legacy behaviour (just ack)
	return &EnqueueAck{MemoryID: memID, Status: "created"}, nil
}

// ListEntriesInVault lists entries within a memory using vaultId.
func (c *Client) ListEntriesInVault(ctx context.Context, userID, vaultID, memID string, params map[string]string) (*ListEntriesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := requireUserID(userID); err != nil {
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
	url := fmt.Sprintf("%s/api/users/%s/vaults/%s/memories/%s/entries%s", c.baseURL, userID, vaultID, memID, query)
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
		return nil, fmt.Errorf("list entries: status %d", resp.StatusCode)
	}

	var lr ListEntriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}
