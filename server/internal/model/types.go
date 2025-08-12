package model

import "time"

// User represents an account in the system.
type User struct {
	UserID         string     `json:"userId"`
	Email          string     `json:"email"`
	DisplayName    *string    `json:"displayName,omitempty"`
	TimeZone       string     `json:"timeZone"`
	Status         string     `json:"status"`
	CreationTime   time.Time  `json:"creationTime"`
	LastActiveTime *time.Time `json:"lastActiveTime,omitempty"`
}

// Vault groups memories under a user.
type Vault struct {
	VaultID      string    `json:"vaultId"`
	UserID       string    `json:"userId"`
	Title        string    `json:"title"`
	CreationTime time.Time `json:"creationTime"`
}

// Memory is a container for entries and contexts.
type Memory struct {
	MemoryID     string    `json:"memoryId"`
	UserID       string    `json:"userId"`
	VaultID      string    `json:"vaultId"`
	MemoryType   string    `json:"memoryType"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// MemoryEntry is an immutable record of content with optional summary and metadata.
type MemoryEntry struct {
	EntryID        string                 `json:"entryId"`
	UserID         string                 `json:"userId"`
	VaultID        string                 `json:"vaultId"`
	MemoryID       string                 `json:"memoryId"`
	RawEntry       string                 `json:"rawEntry"`
	Summary        *string                `json:"summary,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Tags           map[string]interface{} `json:"tags,omitempty"`
	CreationTime   time.Time              `json:"creationTime"`
	ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
}

// MemoryContext stores the latest context snapshot for a memory.
type MemoryContext struct {
	ContextID    string    `json:"contextId"`
	UserID       string    `json:"userId"`
	VaultID      string    `json:"vaultId"`
	MemoryID     string    `json:"memoryId"`
	ContextJSON  []byte    `json:"context"`
	CreationTime time.Time `json:"creationTime"`
}

// SearchHit represents a search result from the index.
type SearchHit struct {
	EntryID string  `json:"entryId"`
	Summary string  `json:"summary"`
	Score   float64 `json:"score"`
}

// ListEntriesRequest captures filters used when listing entries.
type ListEntriesRequest struct {
	UserID   string
	VaultID  string
	MemoryID string
	Limit    int
	Before   *time.Time
	After    *time.Time
}
