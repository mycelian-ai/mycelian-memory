package factory

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// NewSearchIndex creates a search index implementation based on config.
// Launches async bootstrap with short timeout; returns index immediately for fast startup.
func NewSearchIndex(ctx context.Context, cfg *config.Config, log zerolog.Logger) (searchindex.Index, error) {
	if cfg.SearchIndexURL == "" {
		return nil, fmt.Errorf("search index URL not configured - required for service operation")
	}

	// Create Weaviate index client
	idx, err := searchindex.NewWaviateNativeIndex(cfg.SearchIndexURL)
	if err != nil {
		return nil, err
	}

	// Async bootstrap with configurable timeout; don't block startup
	go func() {
		bootstrapTimeout := time.Duration(cfg.BootstrapTimeoutSeconds) * time.Second
		bootstrapCtx, cancel := context.WithTimeout(ctx, bootstrapTimeout)
		defer cancel()

		if err := searchindex.BootstrapWaviate(bootstrapCtx, cfg.SearchIndexURL); err != nil {
			log.Warn().Err(err).Str("url", cfg.SearchIndexURL).Msg("search index bootstrap failed")
		} else {
			log.Debug().Str("url", cfg.SearchIndexURL).Msg("search index bootstrap completed")
		}
	}()

	return idx, nil
}
