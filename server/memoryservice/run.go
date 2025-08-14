package memoryservice

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mycelian/mycelian-memory/server/internal/api"
	"github.com/mycelian/mycelian-memory/server/internal/config"
	emb "github.com/mycelian/mycelian-memory/server/internal/embeddings"
	"github.com/mycelian/mycelian-memory/server/internal/factory"
	"github.com/mycelian/mycelian-memory/server/internal/health"
	"github.com/mycelian/mycelian-memory/server/internal/logger"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
	"github.com/mycelian/mycelian-memory/server/internal/services"
	"github.com/mycelian/mycelian-memory/server/internal/store"
	"github.com/rs/zerolog"
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
		Str("search_index_url", cfg.SearchIndexURL).
		Str("embed_provider", cfg.EmbedProvider).
		Str("embed_model", cfg.EmbedModel).
		Msg("Memory service starting")

	// Create cancellable root context bound to SIGINT/SIGTERM
	ctx, stop := newServerContext()
	defer stop()

	// Initialize dependencies (store, index, embedder)
	st, idx, embedProvider, err := initDependencies(ctx, cfg, log)
	if err != nil {
		return err
	}

	// Build router
	router := buildRouter(st, idx, embedProvider, cfg, log)

	// Start health checkers and bind service health
	svcHealth := startHealthCheckers(ctx, cfg, log, st, idx, embedProvider)

	// Block startup until dependencies report healthy; fail fast otherwise
	if err := waitUntilHealthy(ctx, cfg, svcHealth); err != nil {
		log.Error().Stack().Err(err).Msg("startup health check failed")
		return err
	}

	// HTTP server and serve
	server := newHTTPServer(ctx, cfg, router)
	errCh := serveHTTP(server, log, cfg)

	// Graceful shutdown on context cancel or server error
	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down server")
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctxShutdown); err != nil {
			log.Error().Stack().Err(err).Msg("Server forced to shutdown")
			return err
		}
		log.Info().Msg("Server exited")
		return nil
	case err := <-errCh:
		log.Error().Stack().Err(err).Msg("HTTP server failed")
		return err
	}
}

// initDependencies constructs required components and enforces fail-fast on missing deps.
func initDependencies(ctx context.Context, cfg *config.Config, log zerolog.Logger) (store.Store, searchindex.Index, emb.EmbeddingProvider, error) {
	st, err := factory.NewStore(ctx, cfg, log)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Store adapter unavailable")
		return nil, nil, nil, err
	}

	idx, err := factory.NewSearchIndex(ctx, cfg, log)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Search index adapter unavailable")
		return nil, nil, nil, err
	}

	embProvider := factory.NewEmbeddingProvider(ctx, cfg, log)
	if embProvider == nil {
		return nil, nil, nil, fmt.Errorf("embedding provider not configured")
	}
	return st, idx, embProvider, nil
}

// buildRouter wires HTTP routes to handlers.
func buildRouter(st store.Store, idx searchindex.Index, embProvider emb.EmbeddingProvider, cfg *config.Config, log zerolog.Logger) *mux.Router {
	root := mux.NewRouter()
	root.Use(api.Recover)

	// Users
	userSvc := services.NewUserService(st)
	userHandler := api.NewUserHandler(userSvc)
	root.HandleFunc("/api/users", userHandler.CreateUser).Methods("POST")
	root.HandleFunc("/api/users/{userId}", userHandler.GetUser).Methods("GET")

	// Vaults
	vaultSvc := services.NewVaultService(st, idx)
	vault := api.NewVaultHandler(vaultSvc)
	root.HandleFunc("/api/users/{userId}/vaults", vault.CreateVault).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults", vault.ListVaults).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}", vault.GetVault).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}", vault.DeleteVault).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach", vault.AttachMemoryToVault).Methods("POST")

	// Memories
	memorySvc := services.NewMemoryService(st, idx, embProvider)
	memory := api.NewMemoryHandler(memorySvc, vaultSvc)
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories", memory.CreateMemory).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories", memory.ListMemories).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}", memory.GetMemory).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}", memory.DeleteMemory).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries", memory.ListMemoryEntries).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries", memory.CreateMemoryEntry).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}", memory.GetMemoryEntryByID).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}", memory.DeleteMemoryEntryByID).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}/tags", memory.UpdateMemoryEntryTags).Methods("PATCH")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts", memory.PutMemoryContext).Methods("PUT")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts", memory.GetLatestMemoryContext).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}", memory.DeleteMemoryContextByID).Methods("DELETE")

	// Title-based
	root.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories", memory.ListMemoriesByVaultTitle).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}", memory.GetMemoryByTitle).Methods("GET")

	// Health
	healthHandler := api.NewHealthHandler()
	root.HandleFunc("/api/health", healthHandler.CheckHealth).Methods("GET")

	// Search
	search, err := api.NewSearchHandler(embProvider, idx, cfg.SearchAlpha)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to create search handler")
		// Handle gracefully - skip search endpoint registration
	} else {
		root.HandleFunc("/api/search", search.HandleSearch).Methods("POST")
	}
	return root
}

// startHealthCheckers starts component checkers and service-level aggregator; binds health.
func startHealthCheckers(ctx context.Context, cfg *config.Config, log zerolog.Logger, st store.Store, idx searchindex.Index, embProvider emb.EmbeddingProvider) *health.ServiceHealthChecker {
	var checkers []health.HealthChecker
	probeTimeout := time.Duration(cfg.HealthProbeTimeoutSeconds) * time.Second
	interval := time.Duration(cfg.HealthIntervalSeconds) * time.Second

	storeChecker := store.NewStoreHealthChecker(st, log, probeTimeout)
	go storeChecker.Start(ctx, interval)
	checkers = append(checkers, storeChecker)

	idxChecker := searchindex.NewSearchIndexHealthChecker(idx, log, probeTimeout)
	go idxChecker.Start(ctx, interval)
	checkers = append(checkers, idxChecker)

	embChecker := emb.NewProviderHealthChecker(embProvider, log, probeTimeout)
	go embChecker.Start(ctx, interval)
	checkers = append(checkers, embChecker)

	svcHealth := health.NewServiceHealthChecker(log, checkers...)
	go svcHealth.Start(ctx, interval)
	api.BindServiceHealth(svcHealth.IsHealthy)
	return svcHealth
}

func newHTTPServer(ctx context.Context, cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
}

func serveHTTP(server *http.Server, log zerolog.Logger, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		log.Info().Int("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	return errCh
}

// calculateStartupHealthTimeout returns the startup health timeout in seconds,
// calculated as interval*2 with a minimum of 60 seconds.
func calculateStartupHealthTimeout(healthIntervalSeconds int) int {
	timeout := healthIntervalSeconds * 2
	if timeout < 60 {
		return 60
	}
	return timeout
}

// waitUntilHealthy blocks until service health is healthy or the startup window expires.
func waitUntilHealthy(ctx context.Context, cfg *config.Config, svcHealth *health.ServiceHealthChecker) error {
	// Allow extra time for health checkers to complete their first probe cycle
	// Health checkers start as unhealthy and need time to run their first check
	timeoutSeconds := calculateStartupHealthTimeout(cfg.HealthIntervalSeconds)
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		if svcHealth.IsHealthy() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("startup aborted: dependencies not healthy within %d seconds", timeoutSeconds)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// newServerContext returns a cancellable context that is cancelled on SIGINT/SIGTERM.
func newServerContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
