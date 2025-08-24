package store

import (
	"context"

	"github.com/mycelian/mycelian-memory/server/internal/model"
)

// Store defines the persistence surface used by the application services.
// It provides typed accessors for each resource area (users, vaults, memories,
// entries, contexts) and hides concrete database details behind simple
// method contracts. Drivers (e.g., Postgres) live under
// internal/store/<driver>/ and implement these interfaces.
//
// Goals:
// - Keep business logic free of SQL/driver specifics
// - Centralize data validation and not-found handling
// - Provide clear, minimal methods for the operations the app needs
// - Make it straightforward to test services using mocks
type Store interface {
	Users() Users
	Vaults() Vaults
	Memories() Memories
	Entries() Entries
	Contexts() Contexts
}

type Users interface {
	Create(ctx context.Context, u *model.User) (*model.User, error)
	Get(ctx context.Context, userID string) (*model.User, error)
	Delete(ctx context.Context, userID string) error
}

type Vaults interface {
	Create(ctx context.Context, v *model.Vault) (*model.Vault, error)
	GetByID(ctx context.Context, userID, vaultID string) (*model.Vault, error)
	GetByTitle(ctx context.Context, userID, title string) (*model.Vault, error)
	List(ctx context.Context, userID string) ([]*model.Vault, error)
	Delete(ctx context.Context, userID, vaultID string) error
	AddMemory(ctx context.Context, userID, vaultID, memoryID string) error
}

type Memories interface {
	Create(ctx context.Context, m *model.Memory) (*model.Memory, error)
	GetByID(ctx context.Context, userID, vaultID, memoryID string) (*model.Memory, error)
	GetByTitle(ctx context.Context, userID, vaultID, title string) (*model.Memory, error)
	List(ctx context.Context, userID, vaultID string) ([]*model.Memory, error)
	Delete(ctx context.Context, userID, vaultID, memoryID string) error
}

type Entries interface {
	Create(ctx context.Context, e *model.MemoryEntry) (*model.MemoryEntry, error)
	List(ctx context.Context, req model.ListEntriesRequest) ([]*model.MemoryEntry, error)
	GetByID(ctx context.Context, userID, vaultID, memoryID, entryID string) (*model.MemoryEntry, error)
	UpdateTags(ctx context.Context, userID, vaultID, memoryID, entryID string, tags map[string]interface{}) (*model.MemoryEntry, error)
	DeleteByID(ctx context.Context, userID, vaultID, memoryID, entryID string) error
}

type Contexts interface {
	Put(ctx context.Context, c *model.MemoryContext) (*model.MemoryContext, error)
	Latest(ctx context.Context, userID, vaultID, memoryID string) (*model.MemoryContext, error)
	DeleteByID(ctx context.Context, userID, vaultID, memoryID, contextID string) error
}
