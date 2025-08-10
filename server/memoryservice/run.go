package memoryservice

import (
	"context"
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

// Run starts the memory service HTTP server and blocks until shutdown or error.
func Run() error {
	log := logger.New("memory-service")

	cfg, err := config.New()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return err
	}

	log.Info().
		Str("build_target", cfg.BuildTarget).
		Str("db_driver", cfg.DBDriver).
		Int("http_port", cfg.HTTPPort).
		Msg("Memory service starting")

	// Initialize storage layer
	ctx := context.Background()
	storageLayer, err := factory.NewStorage(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Storage adapter unavailable")
		return err
	}

	// Start background health monitor
	httpHandlers.StartHealthMonitor(ctx, storageLayer, cfg.WaviateURL, cfg.EmbedProvider, cfg.EmbedModel, 30*time.Second)

	// Configure HTTP router and server
	router := api.NewRouter(storageLayer)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Int("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Graceful shutdown on signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("Shutting down server")
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctxShutdown); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
			return err
		}
		log.Info().Msg("Server exited")
		return nil
	case err := <-errCh:
		log.Error().Err(err).Msg("HTTP server failed")
		return err
	}
}
