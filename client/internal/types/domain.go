package types

import "time"

// ------------------------------
// Core Domain Entities
// ------------------------------

// User represents a user
type User struct {
	ID          string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName,omitempty"`
	TimeZone    string    `json:"timeZone,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Vault represents a vault
type Vault struct {
	UserID       string    `json:"actorId"`
	VaultID      string    `json:"vaultId"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// Memory represents a memory
type Memory struct {
	ID          string    `json:"memoryId"`
	VaultID     string    `json:"vaultId"`
	UserID      string    `json:"actorId"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	MemoryType  string    `json:"memoryType"`
	CreatedAt   time.Time `json:"creationTime"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Entry represents an entry
type Entry struct {
	ID             string            `json:"entryId"`
	UserID         string            `json:"actorId"`
	MemoryID       string            `json:"memoryId"`
	VaultID        string            `json:"vaultId"`
	CreationTime   time.Time         `json:"creationTime"`
	RawEntry       string            `json:"rawEntry"`
	Summary        string            `json:"summary,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	ExpirationTime *time.Time        `json:"expirationTime,omitempty"`
}

// Context represents a context snapshot
type Context struct {
	ContextID    string    `json:"contextId"`
	MemoryID     string    `json:"memoryId"`
	VaultID      string    `json:"vaultId"`
	UserID       string    `json:"actorId"`
	CreationTime time.Time `json:"creationTime"`
	Context      any       `json:"context"`
}
