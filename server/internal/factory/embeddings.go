package factory

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	emb "github.com/mycelian/mycelian-memory/server/internal/embeddings"
	"github.com/mycelian/mycelian-memory/server/internal/embeddings/ollama"
)

// NewEmbeddingProvider creates an embedding provider based on config.
// Launches optional async warmup; returns provider immediately for fast startup.
func NewEmbeddingProvider(ctx context.Context, cfg *config.Config, log zerolog.Logger) emb.EmbeddingProvider {
	var provider emb.EmbeddingProvider

	switch cfg.EmbedProvider {
	case "", "ollama":
		provider = ollama.New(cfg.EmbedModel)
	default:
		log.Warn().Str("provider", cfg.EmbedProvider).Msg("unknown embedding provider; using ollama")
		provider = ollama.New(cfg.EmbedModel)
	}

	if provider == nil {
		return nil
	}

	// Optional async warmup with configurable timeout; don't block startup
	go func() {
		warmupTimeout := time.Duration(cfg.BootstrapTimeoutSeconds) * time.Second
		warmupCtx, cancel := context.WithTimeout(ctx, warmupTimeout)
		defer cancel()

		if vec, err := provider.Embed(warmupCtx, "factory-warmup-check"); err != nil || len(vec) == 0 {
			log.Warn().Err(err).Int("vec_len", len(vec)).
				Str("provider", cfg.EmbedProvider).Str("model", cfg.EmbedModel).
				Msg("embedding provider warmup failed")
		} else {
			log.Debug().Str("provider", cfg.EmbedProvider).Str("model", cfg.EmbedModel).
				Msg("embedding provider warmup completed")
		}
	}()

	return provider
}
