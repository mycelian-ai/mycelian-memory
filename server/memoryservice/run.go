package memoryservice

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	apihttp "github.com/mycelian/mycelian-memory/server/internal/api/http"
	"github.com/mycelian/mycelian-memory/server/internal/api/recovery"
	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/embeddings/ollama"
	"github.com/mycelian/mycelian-memory/server/internal/platform/factory"
	"github.com/mycelian/mycelian-memory/server/internal/platform/logger"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
	"github.com/mycelian/mycelian-memory/server/internal/services"
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

	// Initialize store layer
	ctx := context.Background()
	newStore, err := factory.NewStore(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Store adapter unavailable")
		return err
	}

	// Start background health monitor
	apihttp.StartHealthMonitor(ctx, cfg.WaviateURL, cfg.EmbedProvider, cfg.EmbedModel, 30*time.Second)

	// Configure HTTP router and server
	// Build a root router where migrated endpoints are registered first,
	// then mount the legacy router for everything else.
	root := mux.NewRouter()
	root.Use(recovery.Middleware)

	// No legacy router; wire v2 routes directly

	// New thin API: wire CreateUser and GetUser via the new service path
	userSvc := services.NewUserService(newStore)
	userHandler := apihttp.NewUserHandler(userSvc)
	root.HandleFunc("/api/users", userHandler.CreateUser).Methods("POST")
	root.HandleFunc("/api/users/{userId}", userHandler.GetUser).Methods("GET")

	// Vault v2 handlers wired to the new services
	// Note: pass the search index so vault deletes can propagate to the index.
	// memIdx is initialized below and may be nil if Waviate is not configured.
	var memIdx searchindex.Index
	// Embeddings: switch to native embeddings package
	if cfg.WaviateURL != "" {
		// In dev/e2e, ensure schema exists before creating the index client.
		// Bootstrap is idempotent and safe under race.
		_ = searchindex.BootstrapWaviate(ctx, cfg.WaviateURL)
		if idx, err := searchindex.NewWaviateNativeIndex(cfg.WaviateURL); err == nil {
			memIdx = idx
		}
	}
	// For now, support only ollama provider as a stub; expand with openai in follow-up
	var embedding interface {
		Embed(context.Context, string) ([]float32, error)
	}
	// Single embedder implementation: Ollama. The model name (e.g., "mxbai-embed-large",
	// "nomic-embed-text") is passed through to Ollama.
	if cfg.EmbedProvider == "ollama" || cfg.EmbedProvider == "" {
		embedding = ollama.New(cfg.EmbedModel)
	}
	// Validate embedder readiness at startup: must return a non-empty vector
	if embedding != nil {
		if vec, err := embedding.Embed(context.Background(), "service-startup-check"); err != nil || len(vec) == 0 {
			return fmt.Errorf("embedder not ready: provider=%s model=%s err=%v len=%d", cfg.EmbedProvider, cfg.EmbedModel, err, len(vec))
		}
	}

	vaultSvc := services.NewVaultService(newStore, memIdx)
	vaultV2 := apihttp.NewVaultV2Handler(vaultSvc)
	root.HandleFunc("/api/users/{userId}/vaults", vaultV2.CreateVault).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults", vaultV2.ListVaults).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}", vaultV2.GetVault).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}", vaultV2.DeleteVault).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach", vaultV2.AttachMemoryToVault).Methods("POST")

	// Memory v2 handlers (wire SearchIndex + Embeddings when configured)
	memorySvc := services.NewMemoryService(newStore, memIdx, embedding)
	memoryV2 := apihttp.NewMemoryV2Handler(memorySvc, vaultSvc)
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories", memoryV2.CreateMemory).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories", memoryV2.ListMemories).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}", memoryV2.GetMemory).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}", memoryV2.DeleteMemory).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries", memoryV2.ListMemoryEntries).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries", memoryV2.CreateMemoryEntry).Methods("POST")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}", memoryV2.GetMemoryEntryByID).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}", memoryV2.DeleteMemoryEntryByID).Methods("DELETE")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}/tags", memoryV2.UpdateMemoryEntryTags).Methods("PATCH")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts", memoryV2.PutMemoryContext).Methods("PUT")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts", memoryV2.GetLatestMemoryContext).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}", memoryV2.DeleteMemoryContextByID).Methods("DELETE")

	// Title-based routes
	root.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories", memoryV2.ListMemoriesByVaultTitle).Methods("GET")
	root.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}", memoryV2.GetMemoryByTitle).Methods("GET")

	// Health (v2)
	healthHandler := apihttp.NewHealthHandler()
	root.HandleFunc("/api/health", healthHandler.CheckHealth).Methods("GET")

	// Search (v2 – native index + embeddings)
	var embProv interface {
		Embed(context.Context, string) ([]float32, error)
	}
	if cfg.EmbedProvider == "ollama" || cfg.EmbedProvider == "" {
		embProv = ollama.New(cfg.EmbedModel)
	}
	searchV2 := apihttp.NewSearchV2Handler(embProv, memIdx, cfg.SearchAlpha)
	root.HandleFunc("/api/search", searchV2.HandleSearch).Methods("POST")

	// No legacy catch-all
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      root,
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
