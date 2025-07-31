package client

import (
	"time"
)

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
