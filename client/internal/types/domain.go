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
	UserID       string    `json:"userId"`
	VaultID      string    `json:"vaultId"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// Memory represents a memory
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

// Entry represents an entry
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
