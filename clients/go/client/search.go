package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SearchRequest payload for POST /api/search.
// Note: the JSON field names follow the backend spec exactly.
// UserID and MemoryID are required to enforce tenant isolation.
// TopK is optional (defaults to 10 when zero).
//
// Example:
//
//	req := client.SearchRequest{
//	    UserID:   userID,
//	    MemoryID: memID,
//	    Query:    "kubernetes",
//	    TopK:     5,
//	}
//
//	resp, err := cli.Search(ctx, req)
//
// Both search entries *and* the latest context string are returned.
// See SearchResponse for the full schema.
type SearchRequest struct {
	UserID   string `json:"userId"`
	MemoryID string `json:"memoryId"`
	Query    string `json:"query"`
	TopK     int    `json:"topK,omitempty"`
}

// SearchEntry represents one hit in the search response.  It mirrors a
// MemoryEntry subset plus a relevance score.
type SearchEntry struct {
	Entry
	Score float64 `json:"score"`
}

// SearchResponse wraps the /api/search result.
type SearchResponse struct {
	Entries              []SearchEntry   `json:"entries"`
	Count                int             `json:"count"`
	LatestContext        json.RawMessage `json:"latestContext,omitempty"`
	ContextTimestamp     *time.Time      `json:"contextTimestamp,omitempty"`
	BestContext          json.RawMessage `json:"bestContext,omitempty"`
	BestContextTimestamp *time.Time      `json:"bestContextTimestamp,omitempty"`
	BestContextScore     *float64        `json:"bestContextScore,omitempty"`
}

// Search executes a hybrid semantic/keyword search constrained to a single
// memory. It performs a blocking HTTP call and returns the top-K matches along
// with the latest contextual document stored for the memory.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/search", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: status %d", resp.StatusCode)
	}

	var sr SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}
