package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"database/sql"

	"github.com/rs/zerolog/log"

	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/outbox"
	"github.com/mycelian/mycelian-memory/server/internal/search"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	// deps
	db, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres open")
	}
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("postgres ping")
	}

	emb, err := search.NewProvider(cfg.EmbedProvider, cfg.EmbedModel)
	if err != nil {
		log.Warn().Err(err).Msg("embedder unavailable – vectors will be empty")
		emb = nil
	}

	// Ensure schema is bootstrapped with multi-tenancy enabled before any writes
	if err := search.BootstrapWaviate(context.Background(), cfg.WaviateURL); err != nil {
		log.Error().Err(err).Msg("waviate bootstrap")
	}

	wav, err := search.NewWaviateSearcher(cfg.WaviateURL)
	if err != nil {
		log.Fatal().Err(err).Msg("waviate")
	}

	w := outbox.NewWorker(db, emb, wav, outbox.Config{
		PostgresDSN: cfg.PostgresDSN,
		BatchSize:   100,
		Interval:    2 * time.Second,
	}, log.Logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Error().Err(err).Msg("outbox worker exit")
		os.Exit(1)
	}
}
