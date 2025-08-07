package factory

import (
	"context"
	"fmt"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/localstate"
	"github.com/mycelian/mycelian-memory/server/internal/platform/database"
	"github.com/mycelian/mycelian-memory/server/internal/storage"
	"github.com/mycelian/mycelian-memory/server/internal/storage/sqlite"
)

// NewStorage selects the correct storage adapter based on cfg.DBDriver.
// Adapter constructors will be wired in follow-up tasks; for non-implemented
// drivers, this function currently returns an explicit TODO error.
func NewStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.DBDriver {
	case "spanner-pg":
		// Use Cloud Spanner client in PostgreSQL dialect mode. The emulator or real
		// instance is pointed to via SPANNER_EMULATOR_HOST or production endpoint.
		client, err := database.NewSpannerClient(context.Background(), database.SpannerConfig{
			ProjectID:  cfg.GCPProjectID,
			InstanceID: cfg.SpannerInstanceID,
			DatabaseID: cfg.SpannerDatabaseID,
		})
		if err != nil {
			return nil, err
		}
		return storage.NewSpannerStorage(client), nil
	case "postgres":
		return nil, fmt.Errorf("TODO: postgres adapter not implemented")
	case "sqlite":
		db, err := sqlite.Open(cfg.SQLitePath)
		if err != nil {
			return nil, err
		}
		if err := localstate.EnsureSQLiteSchema(db); err != nil {
			return nil, err
		}
		return sqlite.NewSqliteStorageWithDB(db)
	default:
		return nil, fmt.Errorf("unknown DB_DRIVER: %s", cfg.DBDriver)
	}
}
