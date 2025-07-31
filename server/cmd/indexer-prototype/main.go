package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	indexer "memory-backend/internal/indexer-prototype"
	platformlogger "memory-backend/internal/platform/logger"
)

func main() {
	// Load configuration from flags/env
	cfg := indexer.Load()

	// Initialise logger with service name
	logger := platformlogger.New("indexer-prototype")

	// Replace global logger for convenience
	log.Logger = logger

	// Create embedder provider
	emb, err := indexer.NewProvider(cfg.Provider, cfg.EmbedModel)
	if err != nil {
		logger.Warn().Err(err).Msg("embedder unavailable – proceeding with nil embedder (vectors will be empty)")
		emb = nil
	}

	// Build supporting components
	var scanner indexer.Scanner
	switch cfg.DBDriver {
	case "sqlite":
		scanner, err = indexer.NewSQLiteScanner(cfg.SQLitePath, logger)
	default:
		scanner, err = indexer.NewScanner(context.Background(), cfg, logger) // Spanner
	}
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init scanner")
	}
	uploader, err := indexer.NewUploader(cfg.WeaviateURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init uploader")
	}
	state, err := indexer.NewState("")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init state store")
	}

	// Build indexer instance
	idx := indexer.New(cfg, emb, scanner, uploader, state, logger)

	// Create a context that will be cancelled on shutdown signals.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the indexer in a separate goroutine.
	go func() {
		if err := idx.Run(ctx); err != nil && err != context.Canceled {
			logger.Error().Err(err).Msg("indexer terminated with error")
			os.Exit(1)
		}
	}()

	// Wait for the context to be cancelled (e.g. by a shutdown signal).
	<-ctx.Done()

	logger.Info().Msg("shutting down main")
}
