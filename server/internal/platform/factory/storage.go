package factory

import (
	"fmt"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	storepkg "github.com/mycelian/mycelian-memory/server/internal/store"
	storepg "github.com/mycelian/mycelian-memory/server/internal/store/postgres"
)

// NewStorage selects the legacy storage adapter for components that still need it.
// This will be removed after full migration to the new store path.
// NewStorage deprecated: callers should use NewStore. Kept temporarily for tests.
// func NewStorage(cfg *config.Config) (storage.Storage, error) { return nil, fmt.Errorf("deprecated") }

// NewStore returns a store.Store backed by Postgres using the legacy storage adapter underneath.
// This bridges us to the new hexagonal store interface without legacy wiring at call sites.
func NewStore(cfg *config.Config) (storepkg.Store, error) {
	if cfg.DBDriver != "postgres" {
		return nil, fmt.Errorf("unknown DB_DRIVER: %s", cfg.DBDriver)
	}
	dsn := cfg.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("MEMORY_BACKEND_POSTGRES_DSN is required when DB_DRIVER=postgres")
	}
	db, err := storepg.Open(dsn)
	if err != nil {
		return nil, err
	}
	return storepg.NewWithDB(db), nil
}
