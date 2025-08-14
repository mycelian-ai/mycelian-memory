package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	storepkg "github.com/mycelian/mycelian-memory/server/internal/store"
	storepg "github.com/mycelian/mycelian-memory/server/internal/store/postgres"
)

// NewStore returns a Postgres-backed store.Store.
// Requires cfg.DBDriver == "postgres" and a non-empty cfg.PostgresDSN.
// Launches async bootstrap check; returns store immediately for fast startup.
func NewStore(ctx context.Context, cfg *config.Config, log zerolog.Logger) (storepkg.Store, error) {
	if cfg.DBDriver != "postgres" {
		return nil, fmt.Errorf("unknown DB_DRIVER: %s", cfg.DBDriver)
	}
	dsn := cfg.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("MEMORY_SERVER_POSTGRES_DSN is required when DB_DRIVER=postgres")
	}

	// Open connection synchronously since health checks need it immediately
	db, err := storepg.Open(dsn)
	if err != nil {
		return nil, err
	}

	// Async bootstrap check with configurable timeout; don't block startup
	go func() {
		bootstrapTimeout := time.Duration(cfg.BootstrapTimeoutSeconds) * time.Second
		bootstrapCtx, cancel := context.WithTimeout(ctx, bootstrapTimeout)
		defer cancel()

		if err := storepg.Bootstrap(bootstrapCtx, dsn); err != nil {
			log.Warn().Err(err).Str("driver", cfg.DBDriver).Msg("store bootstrap check failed")
		} else {
			log.Debug().Str("driver", cfg.DBDriver).Msg("store bootstrap check completed")
		}
	}()

	return storepg.NewWithDB(db), nil
}
