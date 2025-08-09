package factory

import (
	"fmt"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
	pgadapter "github.com/mycelian/mycelian-memory/server/internal/storage/postgres"
)

// NewStorage selects the correct storage adapter based on cfg.DBDriver.
// Adapter constructors will be wired in follow-up tasks; for non-implemented
// drivers, this function currently returns an explicit TODO error.
func NewStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.DBDriver {
	case "spanner-pg":
		return nil, fmt.Errorf("spanner adapter removed; use postgres")
	case "postgres":
		// Expect DSN in MEMORY_BACKEND_POSTGRES_DSN
		dsn := cfg.PostgresDSN
		if dsn == "" {
			return nil, fmt.Errorf("MEMORY_BACKEND_POSTGRES_DSN is required when DB_DRIVER=postgres")
		}
		db, err := pgadapter.Open(dsn)
		if err != nil {
			return nil, err
		}
		return pgadapter.NewPostgresStorageWithDB(db)
	default:
		return nil, fmt.Errorf("unknown DB_DRIVER: %s", cfg.DBDriver)
	}
}
