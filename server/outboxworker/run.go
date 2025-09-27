package outboxworker

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/rs/zerolog/log"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/embeddings/ollama"
	"github.com/mycelian/mycelian-memory/server/internal/outbox"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// Run starts the outbox worker and blocks until shutdown or error.
func Run() error {
	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres open")
	}
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("postgres ping")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Warn().Err(err).Msg("closing postgres connection")
		}
	}()

	var emb interface {
		Embed(context.Context, string) ([]float32, error)
	}
	if cfg.EmbedProvider == "ollama" || cfg.EmbedProvider == "" {
		emb = ollama.New(cfg.EmbedModel)
	}
	// Critical dependency check - fail fast if embedder is missing
	if emb == nil {
		log.Fatal().Str("provider", cfg.EmbedProvider).Msg("critical dependency missing: embedder not configured")
	}
	// Validate embedder readiness at startup
	startupTimeout := time.Duration(cfg.OutboxStartupEmbedTimeoutSeconds) * time.Second
	if startupTimeout <= 0 {
		startupTimeout = 10 * time.Second
	}
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), startupTimeout)
	defer cancelStartup()
	if vec, err := emb.Embed(startupCtx, "worker-startup-check"); err != nil || len(vec) == 0 {
		return fmt.Errorf("embedder not ready: provider=%s model=%s err=%v len=%d", cfg.EmbedProvider, cfg.EmbedModel, err, len(vec))
	}

	// Ensure schema exists in dev/e2e; safe to call repeatedly.
	bootstrapTimeout := time.Duration(cfg.BootstrapTimeoutSeconds) * time.Second
	if bootstrapTimeout <= 0 {
		bootstrapTimeout = 5 * time.Second
	}
	bootstrapCtx, cancelBootstrap := context.WithTimeout(context.Background(), bootstrapTimeout)
	defer cancelBootstrap()
	if err := searchindex.BootstrapWeaviate(bootstrapCtx, cfg.SearchIndexURL); err != nil {
		return fmt.Errorf("bootstrap search index: %w", err)
	}
	idx, err := searchindex.NewWeaviateNativeIndex(cfg.SearchIndexURL)
	if err != nil {
		log.Fatal().Err(err).Msg("search index")
	}

	w := outbox.NewWorker(db, emb, idx, outbox.Config{
		PostgresDSN:  cfg.PostgresDSN,
		BatchSize:    cfg.OutboxBatchSize,
		Interval:     time.Duration(cfg.OutboxIntervalSeconds) * time.Second,
		EmbedTimeout: time.Duration(cfg.OutboxEmbedTimeoutSeconds) * time.Second,
		IndexTimeout: time.Duration(cfg.OutboxIndexTimeoutSeconds) * time.Second,
		BackoffCap:   time.Duration(cfg.OutboxBackoffCapSeconds) * time.Second,
	}, log.Logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Error().Err(err).Msg("outbox worker exit")
		return err
	}
	return nil
}
