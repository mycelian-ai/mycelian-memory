package factory

import (
	"context"
	"fmt"

	"memory-backend/internal/config"
	"memory-backend/internal/localstate"
	"memory-backend/internal/platform/database"
	"memory-backend/internal/storage"
	"memory-backend/internal/storage/sqlite"
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
