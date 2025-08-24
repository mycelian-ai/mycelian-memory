package memory

import (
	"time"

	"github.com/google/uuid"
)

// CreateUserRequest removed - user management is now external

// CreateMemoryRequest represents a request to create a memory
type CreateMemoryRequest struct {
	VaultID     uuid.UUID
	ActorID     string
	MemoryType  string
	Title       string
	Description *string
}

// CreateMemoryEntryRequest represents a request to create a memory entry
type CreateMemoryEntryRequest struct {
	VaultID        uuid.UUID
	ActorID        string
	MemoryID       string
	RawEntry       string
	Summary        *string
	Metadata       map[string]interface{}
	Tags           map[string]interface{}
	ExpirationTime *time.Time
}

// ListMemoryEntriesRequest represents a request to list memory entries
type ListMemoryEntriesRequest struct {
	VaultID  uuid.UUID
	ActorID  string
	MemoryID string
	Limit    int
	Before   *time.Time
	After    *time.Time
}

// UpdateMemoryEntryTagsRequest represents a request to update memory entry tags
type UpdateMemoryEntryTagsRequest struct {
	VaultID  uuid.UUID
	ActorID  string
	MemoryID string
	EntryID  string
	Tags     map[string]interface{}
}

// CreateMemoryContextRequest represents a request to create a context snapshot for a memory
// Aligns with storage.CreateMemoryContextRequest but at service layer level.
// ContextID optional; generated if absent.
type CreateMemoryContextRequest struct {
	VaultID   uuid.UUID
	ActorID   string
	MemoryID  string
	ContextID *string
	Context   string
}
