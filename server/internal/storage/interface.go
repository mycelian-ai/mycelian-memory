package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User struct and related types removed - user management is now external

// Memory represents a memory instance
type Memory struct {
	ActorID      string    `json:"actorId"`
	VaultID      uuid.UUID `json:"vaultId"`
	MemoryID     string    `json:"memoryId"`
	MemoryType   string    `json:"memoryType"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// MemoryEntry represents an entry in the memory log
type MemoryEntry struct {
	ActorID                    string                 `json:"actorId"`
	VaultID                    uuid.UUID              `json:"vaultId"`
	MemoryID                   string                 `json:"memoryId"`
	CreationTime               time.Time              `json:"creationTime"`
	EntryID                    string                 `json:"entryId"`
	RawEntry                   string                 `json:"rawEntry"`
	Summary                    *string                `json:"summary,omitempty"`
	Metadata                   map[string]interface{} `json:"metadata,omitempty"` // JSON object (immutable)
	Tags                       map[string]interface{} `json:"tags,omitempty"`     // JSON object (mutable)
	CorrectionTime             *time.Time             `json:"correctionTime,omitempty"`
	CorrectedEntryMemoryId     *string                `json:"correctedEntryMemoryId,omitempty"`
	CorrectedEntryCreationTime *time.Time             `json:"correctedEntryCreationTime,omitempty"`
	CorrectionReason           *string                `json:"correctionReason,omitempty"`
	LastUpdateTime             *time.Time             `json:"lastUpdateTime,omitempty"`
	ExpirationTime             *time.Time             `json:"expirationTime,omitempty"`
}

// MemoryContext represents a snapshot of text context for a memory
// Primary key: (ActorID, MemoryID, ContextID)
// CreationTime is the commit timestamp when inserted.
type MemoryContext struct {
	ActorID      string    `json:"actorId"`
	VaultID      uuid.UUID `json:"vaultId"`
	MemoryID     string    `json:"memoryId"`
	ContextID    string    `json:"contextId"`
	Context      string    `json:"context"`
	CreationTime time.Time `json:"creationTime"`
}

// CreateUserRequest removed - user management is now external

// CreateMemoryRequest represents the request to create a new memory
type CreateMemoryRequest struct {
	VaultID     uuid.UUID `json:"vaultId"`
	ActorID     string    `json:"actorId"`
	MemoryType  string    `json:"memoryType"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
}

// CreateMemoryEntryRequest represents the request to create a new memory entry
type CreateMemoryEntryRequest struct {
	VaultID        uuid.UUID              `json:"vaultId"`
	ActorID        string                 `json:"actorId"`
	MemoryID       string                 `json:"memoryId"`
	RawEntry       string                 `json:"rawEntry"`
	Summary        *string                `json:"summary,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Tags           map[string]interface{} `json:"tags,omitempty"`
	ExpirationTime *time.Time             `json:"expirationTime,omitempty"`
}

// ListMemoryEntriesRequest represents the request to list memory entries
type ListMemoryEntriesRequest struct {
	VaultID  uuid.UUID  `json:"vaultId"`
	ActorID  string     `json:"actorId"`
	MemoryID string     `json:"memoryId"`
	Limit    int        `json:"limit,omitempty"`  // Max entries to return
	Before   *time.Time `json:"before,omitempty"` // Entries before this timestamp
	After    *time.Time `json:"after,omitempty"`  // Entries after this timestamp
}

// CorrectMemoryEntryRequest represents the request to correct a memory entry
type CorrectMemoryEntryRequest struct {
	VaultID              uuid.UUID              `json:"vaultId"`
	ActorID              string                 `json:"actorId"`
	MemoryID             string                 `json:"memoryId"`
	OriginalCreationTime time.Time              `json:"originalCreationTime"`
	CorrectedEntryID     string                 `json:"correctedEntryId"`
	CorrectedContent     string                 `json:"correctedContent"`
	CorrectedSummary     *string                `json:"correctedSummary,omitempty"`
	CorrectionReason     string                 `json:"correctionReason"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	Tags                 map[string]interface{} `json:"tags,omitempty"`
}

// UpdateMemoryEntrySummaryRequest represents the request to update entry summary
type UpdateMemoryEntrySummaryRequest struct {
	VaultID  uuid.UUID `json:"vaultId"`
	ActorID  string    `json:"actorId"`
	MemoryID string    `json:"memoryId"`
	EntryID  string    `json:"entryId"`
	Summary  string    `json:"summary"`
}

// UpdateMemoryEntryTagsRequest represents the request to update entry tags
type UpdateMemoryEntryTagsRequest struct {
	VaultID  uuid.UUID              `json:"vaultId"`
	ActorID  string                 `json:"actorId"`
	MemoryID string                 `json:"memoryId"`
	EntryID  string                 `json:"entryId"`
	Tags     map[string]interface{} `json:"tags"`
}

// CreateMemoryContextRequest represents the request to insert a new context snapshot
// ContextID is optional – if empty, the storage layer should generate a UUID.
type CreateMemoryContextRequest struct {
	VaultID   uuid.UUID `json:"vaultId"`
	ActorID   string    `json:"actorId"`
	MemoryID  string    `json:"memoryId"`
	ContextID *string   `json:"contextId,omitempty"`
	Context   string    `json:"context"`
}

// GetLatestMemoryContextRequest retrieves the most recent context snapshot
// It may be expanded later for versioned retrieval.
// For now we expose convenience params directly in the method signature.

// Vault represents a collection of memories owned by an actor.
type Vault struct {
	ActorID      string    `json:"actorId"`
	VaultID      uuid.UUID `json:"vaultId"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	CreationTime time.Time `json:"creationTime"`
}

// CreateVaultRequest represents the request to create a new vault.
type CreateVaultRequest struct {
	ActorID     string    `json:"actorId"`
	VaultID     uuid.UUID `json:"vaultId"` // Pre-generated by service layer
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
}

// AddMemoryToVaultRequest associates a memory with a vault.
type AddMemoryToVaultRequest struct {
	ActorID  string    `json:"actorId"`
	VaultID  uuid.UUID `json:"vaultId"`
	MemoryID string    `json:"memoryId"`
}

// DeleteMemoryFromVaultRequest removes a memory association from a vault.
type DeleteMemoryFromVaultRequest struct {
	ActorID  string    `json:"actorId"`
	VaultID  uuid.UUID `json:"vaultId"`
	MemoryID string    `json:"memoryId"`
}

// Storage interface defines the contract for memory storage operations
// User management methods removed - user_id is now treated as opaque string identifier
type Storage interface {
	// Memory operations
	CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error)
	GetMemory(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string) (*Memory, error)
	ListMemories(ctx context.Context, actorID string, vaultID uuid.UUID) ([]*Memory, error)
	DeleteMemory(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string) error

	// Memory entry operations
	CreateMemoryEntry(ctx context.Context, req CreateMemoryEntryRequest) (*MemoryEntry, error)
	GetMemoryEntry(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string, creationTime time.Time) (*MemoryEntry, error)
	// GetMemoryEntryByID retrieves a single entry by its external entryId
	GetMemoryEntryByID(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string, entryID string) (*MemoryEntry, error)
	ListMemoryEntries(ctx context.Context, req ListMemoryEntriesRequest) ([]*MemoryEntry, error)
	CorrectMemoryEntry(ctx context.Context, req CorrectMemoryEntryRequest) (*MemoryEntry, error)
	UpdateMemoryEntrySummary(ctx context.Context, req UpdateMemoryEntrySummaryRequest) (*MemoryEntry, error)
	UpdateMemoryEntryTags(ctx context.Context, req UpdateMemoryEntryTagsRequest) (*MemoryEntry, error)
	// Hard delete by external entryId
	DeleteMemoryEntryByID(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string, entryID string) error

	// Memory context operations
	CreateMemoryContext(ctx context.Context, req CreateMemoryContextRequest) (*MemoryContext, error)
	GetLatestMemoryContext(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string) (*MemoryContext, error)
	// Hard delete context snapshot by contextId
	DeleteMemoryContextByID(ctx context.Context, actorID string, vaultID uuid.UUID, memoryID string, contextID string) error

	// Vault operations
	CreateVault(ctx context.Context, req CreateVaultRequest) (*Vault, error)
	GetVault(ctx context.Context, actorID string, vaultID uuid.UUID) (*Vault, error)
	// GetVaultByTitle retrieves a vault by its unique title within an actor scope.
	GetVaultByTitle(ctx context.Context, actorID string, title string) (*Vault, error)
	ListVaults(ctx context.Context, actorID string) ([]*Vault, error)
	DeleteVault(ctx context.Context, actorID string, vaultID uuid.UUID) error

	// Memory lookup by title (unique within vault)
	GetMemoryByTitle(ctx context.Context, actorID string, vaultID uuid.UUID, title string) (*Memory, error)

	AddMemoryToVault(ctx context.Context, req AddMemoryToVaultRequest) error
	DeleteMemoryFromVault(ctx context.Context, req DeleteMemoryFromVaultRequest) error

	// Health check
	HealthCheck(ctx context.Context) error
}
