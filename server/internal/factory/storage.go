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
// NewStore creates a Postgres-backed storage.Store using the provided configuration.
// It validates that cfg.DBDriver equals "postgres" and that cfg.PostgresDSN is non-empty,
// opens the database connection synchronously, and starts an asynchronous bootstrap
// check that uses cfg.BootstrapTimeoutSeconds without blocking startup.
// It returns an error if the driver is unsupported, the DSN is missing, or opening the database fails.
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