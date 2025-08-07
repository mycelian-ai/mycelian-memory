package client

import (
	"encoding/json"
	"fmt"
	"time"
)

// ------------------------------
// Core domain types and payloads
// ------------------------------

// User represents a Synapse user.
type User struct {
	ID          string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName,omitempty"`
	TimeZone    string    `json:"timeZone,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateUserRequest sent to Memory service.
type CreateUserRequest struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
}

// Vault represents a collection of memories belonging to a user.
// It maps 1-to-many with memories and is uniquely identified by a server-generated
// UUIDv4 (vaultId) while retaining a human-readable `title` provided by the user.
// All timestamps are RFC-3339 and parsed into time.Time.
type Vault struct {
	UserID       string    `json:"userId"`
	VaultID      string    `json:"vaultId"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// CreateVaultRequest is the payload for POST /api/users/{userId}/vaults.
type CreateVaultRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// Memory represents a Synapse memory context.
type Memory struct {
	ID          string    `json:"memoryId"`
	VaultID     string    `json:"vaultId"`
	UserID      string    `json:"userId"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	MemoryType  string    `json:"memory_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateMemoryRequest holds parameters for new memory.
type CreateMemoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	MemoryType  string `json:"memoryType"`
}

// Entry represents a memory entry.
type Entry struct {
	ID             string            `json:"entryId"`
	UserID         string            `json:"userId"`
	MemoryID       string            `json:"memoryId"`
	VaultID        string            `json:"vaultId"`
	CreationTime   time.Time         `json:"creationTime"`
	RawEntry       string            `json:"rawEntry"`
	Summary        string            `json:"summary,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	ExpirationTime *time.Time        `json:"expirationTime,omitempty"`
}

// AddEntryRequest payload.
type AddEntryRequest struct {
	RawEntry       string                 `json:"rawEntry"`
	Summary        string                 `json:"summary,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Tags           map[string]string      `json:"tags,omitempty"`
	ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
}

// ListEntriesResponse wraps list endpoint response.
type ListEntriesResponse struct {
	Entries []Entry `json:"entries"`
	Count   int     `json:"count"`
}

// EnqueueAck is returned by write-path endpoints that only enqueue the job.
type EnqueueAck struct {
	MemoryID string `json:"memoryId"`
	Status   string `json:"status"`
}

// ------------------------------
// Search types
// ------------------------------

// SearchRequest payload for POST /api/search.
// UserID and MemoryID enforce tenant isolation.
type SearchRequest struct {
	UserID   string `json:"userId"`
	VaultID  string `json:"vaultId,omitempty"` // optional until backend enforces
	MemoryID string `json:"memoryId"`
	Query    string `json:"query"`
	TopK     int    `json:"topK,omitempty"`
}

// SearchEntry mirrors Entry plus a relevance score.
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

// ------------------------------
// Context snapshot types
// ------------------------------

// PutContextRequest stores a context snapshot.
type PutContextRequest struct {
	Context any `json:"context"`
}

// PutContextResponse contains metadata about a stored context.
type PutContextResponse struct {
	UserID       string    `json:"userId"`
	MemoryID     string    `json:"memoryId"`
	ContextID    string    `json:"contextId"`
	CreationTime time.Time `json:"creationTime"`
}

// GetContextResponse contains the context snapshot and metadata.
type GetContextResponse struct {
	PutContextResponse
	Context any `json:"context"`
}

// ------------------------------
// Common errors
// ------------------------------

var ErrNotFound = fmt.Errorf("context snapshot not found")
