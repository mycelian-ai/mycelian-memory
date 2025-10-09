package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	storagepkg "github.com/mycelian/mycelian-memory/server/internal/storage"
	storagepg "github.com/mycelian/mycelian-memory/server/internal/storage/postgres"
)

// NewStore returns a Postgres-backed store.Store.
// Requires cfg.DBDriver == "postgres" and a non-empty cfg.PostgresDSN.
// NewStore creates and returns a Postgres-backed storage.Store configured from cfg.
// 
// NewStore validates that cfg.DBDriver is "postgres" and that cfg.PostgresDSN is set, opens a database connection synchronously, and returns a store backed by that DB. It also launches an asynchronous bootstrap health check with a timeout derived from cfg.BootstrapTimeoutSeconds; bootstrap failures are logged but do not block startup. An error is returned if the driver is unsupported, the DSN is missing, or opening the database fails.
func NewStore(ctx context.Context, cfg *config.Config, log zerolog.Logger) (storagepkg.Store, error) {
	if cfg.DBDriver != "postgres" {
		return nil, fmt.Errorf("unknown DB_DRIVER: %s", cfg.DBDriver)
	}
	dsn := cfg.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("MEMORY_SERVER_POSTGRES_DSN is required when DB_DRIVER=postgres")
	}

	// Open connection synchronously since health checks need it immediately
	db, err := storagepg.Open(dsn)
	if err != nil {
		return nil, err
	}

	// Async bootstrap check with configurable timeout; don't block startup
	go func() {
		bootstrapTimeout := time.Duration(cfg.BootstrapTimeoutSeconds) * time.Second
		bootstrapCtx, cancel := context.WithTimeout(ctx, bootstrapTimeout)
		defer cancel()

		if err := storagepg.Bootstrap(bootstrapCtx, dsn); err != nil {
			log.Warn().Err(err).Str("driver", cfg.DBDriver).Msg("store bootstrap check failed")
		} else {
			log.Debug().Str("driver", cfg.DBDriver).Msg("store bootstrap check completed")
		}
	}()

	return storagepg.NewWithDB(db), nil
}