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

// Vault groups memories under an actor.
type Vault struct {
	VaultID      string    `json:"vaultId"`
	ActorID      string    `json:"actorId"`
	Title        string    `json:"title"`
	CreationTime time.Time `json:"creationTime"`
}

// Memory is a container for entries and contexts.
type Memory struct {
	MemoryID     string    `json:"memoryId"`
	ActorID      string    `json:"actorId"`
	VaultID      string    `json:"vaultId"`
	MemoryType   string    `json:"memoryType"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// MemoryEntry is an immutable record of content with optional summary and metadata.
type MemoryEntry struct {
	EntryID        string                 `json:"entryId"`
	ActorID        string                 `json:"actorId"`
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
	ActorID      string    `json:"actorId"`
	VaultID      string    `json:"vaultId"`
	MemoryID     string    `json:"memoryId"`
	Context      string    `json:"context"`
	CreationTime time.Time `json:"creationTime"`
}

// SearchHit represents a search result from the index.
type SearchHit struct {
	EntryID      string    `json:"entryId"`
	ActorID      string    `json:"actorId"`
	MemoryID     string    `json:"memoryId"`
	Summary      string    `json:"summary"`
	RawEntry     string    `json:"rawEntry"`
	Score        float64   `json:"score"`
	CreationTime time.Time `json:"creationTime"`
}

// ContextHit represents a matching context shard result
type ContextHit struct {
	Context   string    `json:"context"`
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
}

// ListEntriesRequest captures filters used when listing entries.
type ListEntriesRequest struct {
	ActorID  string
	VaultID  string
	MemoryID string
	Limit    int
	Before   *time.Time
	After    *time.Time
}
