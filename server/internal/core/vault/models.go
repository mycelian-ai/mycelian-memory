package vault

import "github.com/google/uuid"

// CreateVaultRequest represents a request to create a vault.
type CreateVaultRequest struct {
	// UserID is the owner of the Vault.
	UserID string
	// Title is a human-readable label for the Vault.
	Title string
	// Description is an optional longer explanation.
	Description *string
}

// AddMemoryToVaultRequest represents a request to associate a Memory with a Vault.
type AddMemoryToVaultRequest struct {
	UserID   string
	VaultID  uuid.UUID
	MemoryID string
}

// DeleteMemoryFromVaultRequest deletes a Memory from a Vault. Implementation should rely on
// database-level cascading deletes and succeed only if no dependent records remain.
// If the Memory has entries or child objects, the storage layer should return an error.
type DeleteMemoryFromVaultRequest struct {
	UserID   string
	VaultID  uuid.UUID
	MemoryID string
}

// ListVaultsRequest represents a request to list all vaults for a user.
type ListVaultsRequest struct {
	UserID string
}

// ListVaultMemoriesRequest represents a request to list all memories within a vault.
type ListVaultMemoriesRequest struct {
	UserID  string
	VaultID uuid.UUID
}
