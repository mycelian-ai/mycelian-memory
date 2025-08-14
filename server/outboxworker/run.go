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

	var emb interface {
		Embed(context.Context, string) ([]float32, error)
	}
	if cfg.EmbedProvider == "ollama" || cfg.EmbedProvider == "" {
		emb = ollama.New(cfg.EmbedModel)
	}
	// Validate embedder readiness at startup
	if emb != nil {
		if vec, err := emb.Embed(context.Background(), "worker-startup-check"); err != nil || len(vec) == 0 {
			return fmt.Errorf("embedder not ready: provider=%s model=%s err=%v len=%d", cfg.EmbedProvider, cfg.EmbedModel, err, len(vec))
		}
	}

	// Ensure schema exists in dev/e2e; safe to call repeatedly.
	_ = searchindex.BootstrapWaviate(context.Background(), cfg.SearchIndexURL)
	idx, err := searchindex.NewWaviateNativeIndex(cfg.SearchIndexURL)
	if err != nil {
		log.Fatal().Err(err).Msg("search index")
	}

	w := outbox.NewWorker(db, emb, idx, outbox.Config{
		PostgresDSN: cfg.PostgresDSN,
		BatchSize:   100,
		Interval:    2 * time.Second,
	}, log.Logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Error().Err(err).Msg("outbox worker exit")
		return err
	}
	return nil
}
