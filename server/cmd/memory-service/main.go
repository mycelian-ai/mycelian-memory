package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/api"
	httpHandlers "github.com/mycelian/mycelian-memory/server/internal/api/http"
	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/platform/factory"
	"github.com/mycelian/mycelian-memory/server/internal/platform/logger"
)

func main() {
	// Optional build-target flag override (local | cloud-dev)
	buildTarget := flag.String("build-target", "", "Override BUILD_TARGET (local, cloud-dev)")
	flag.Parse()

	log := logger.New("memory-service")

	cfg, err := config.New()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	if *buildTarget != "" {
		cfg.BuildTarget = *buildTarget
		if err := cfg.ResolveDefaults(); err != nil {
			log.Fatal().Err(err).Msg("Invalid build-target override")
		}
	}

	log.Info().
		Str("build_target", cfg.BuildTarget).
		Str("db_driver", cfg.DBDriver).
		Int("http_port", cfg.HTTPPort).
		Msg("Memory service starting…")

		// SQLite support removed

	// -------- Storage layer -----------------
	ctx := context.Background()
	storageLayer, err := factory.NewStorage(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Storage adapter unavailable")
	}

	// -------- Health monitor ---------------
	httpHandlers.StartHealthMonitor(ctx, storageLayer, cfg.WaviateURL, cfg.EmbedProvider, cfg.EmbedModel, 30*time.Second)

	// -------- Router & Server --------------
	router := api.NewRouter(storageLayer)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Int("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server…")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	log.Info().Msg("Server exited")
}
