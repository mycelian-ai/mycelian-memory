package client

import "time"

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
