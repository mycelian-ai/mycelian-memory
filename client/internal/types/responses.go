package types

import (
	"encoding/json"
	"time"
)

// ------------------------------
// Response Types
// ------------------------------

// EnqueueAck represents acknowledgment of async operation
type EnqueueAck struct {
	MemoryID string `json:"memoryId"`
	Status   string `json:"status"`
}

// ListEntriesResponse wraps list endpoint response
type ListEntriesResponse struct {
	Entries []Entry `json:"entries"`
	Count   int     `json:"count"`
}

// PutContextResponse contains metadata about a stored context
type PutContextResponse struct {
	UserID       string    `json:"actorId"`
	MemoryID     string    `json:"memoryId"`
	ContextID    string    `json:"contextId"`
	CreationTime time.Time `json:"creationTime"`
}

// GetContextResponse contains the context snapshot and metadata
type GetContextResponse struct {
	PutContextResponse
	Context any `json:"context"`
}

// SearchEntry mirrors Entry plus a relevance score
type SearchEntry struct {
	Entry
	Score float64 `json:"score"`
}

// SearchResponse wraps the /api/search result
type SearchResponse struct {
	Entries              []SearchEntry   `json:"entries"`
	Count                int             `json:"count"`
	LatestContext        json.RawMessage `json:"latestContext,omitempty"`
	ContextTimestamp     *time.Time      `json:"contextTimestamp,omitempty"`
	BestContext          json.RawMessage `json:"bestContext,omitempty"`
	BestContextTimestamp *time.Time      `json:"bestContextTimestamp,omitempty"`
	BestContextScore     *float64        `json:"bestContextScore,omitempty"`
}

// ListMemoriesResponse mirrors the backend list shape
type ListMemoriesResponse struct {
	Memories []Memory `json:"memories"`
	Count    int      `json:"count"`
}

// ListVaultsResponse mirrors the list endpoint response shape
type ListVaultsResponse struct {
	Vaults []Vault `json:"vaults"`
	Count  int     `json:"count"`
}
