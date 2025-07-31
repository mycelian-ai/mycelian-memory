package api

import (
	httpHandlers "memory-backend/internal/api/http"
	"memory-backend/internal/api/recovery"
	"memory-backend/internal/config"
	"memory-backend/internal/core/memory"
	vaultcore "memory-backend/internal/core/vault"
	"memory-backend/internal/search"
	"memory-backend/internal/storage"

	"github.com/gorilla/mux"
)

// NewRouter creates a new HTTP router with all API routes using clean architecture
func NewRouter(storage storage.Storage) *mux.Router {
	router := mux.NewRouter()

	// Global middlewares
	router.Use(recovery.Middleware)

	// Create domain service
	memoryService := memory.NewService(storage)
	vaultService := vaultcore.NewService(storage)

	// Create handlers
	healthHandler := httpHandlers.NewHealthHandler(storage)
	memoryHandler := httpHandlers.NewMemoryHandler(memoryService, vaultService)
	vaultHandler := httpHandlers.NewVaultHandler(vaultService)

	// Shared configuration & search components
	cfg, _ := config.New() // ignore err for now; assumed validated on startup
	emb, _ := search.NewProvider(cfg.EmbedProvider, cfg.EmbedModel)
	wavSearcher, _ := search.NewWaviateSearcher(cfg.WaviateURL)
	searchHandler := httpHandlers.NewSearchHandler(emb, wavSearcher, cfg.SearchAlpha)

	// Health endpoints
	router.HandleFunc("/api/health", healthHandler.CheckHealth).Methods("GET")
	router.HandleFunc("/api/health/db", healthHandler.CheckStorageHealth).Methods("GET")

	// User endpoints
	router.HandleFunc("/api/users", memoryHandler.CreateUser).Methods("POST")
	router.HandleFunc("/api/users/{userId}", memoryHandler.GetUser).Methods("GET")

	// Memory endpoints under vaults (UUID-based)
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories", memoryHandler.CreateMemory).Methods("POST")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories", memoryHandler.ListMemories).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}", memoryHandler.GetMemory).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}", memoryHandler.DeleteMemory).Methods("DELETE")

	// Memory entry endpoints
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}/entries", memoryHandler.CreateMemoryEntry).Methods("POST")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}/entries", memoryHandler.ListMemoryEntries).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}/entries/{creationTime}/tags", memoryHandler.UpdateMemoryEntryTags).Methods("PATCH")

	// Memory context endpoint
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}/contexts", memoryHandler.PutMemoryContext).Methods("PUT")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}/memories/{memoryId:[0-9a-fA-F-]{36}}/contexts", memoryHandler.GetLatestMemoryContext).Methods("GET")

	// Search endpoint
	router.HandleFunc("/api/search", searchHandler.HandleSearch).Methods("POST")

	// Vault endpoints
	router.HandleFunc("/api/users/{userId}/vaults", vaultHandler.CreateVault).Methods("POST")
	router.HandleFunc("/api/users/{userId}/vaults", vaultHandler.ListVaults).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}", vaultHandler.GetVault).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultId:[0-9a-fA-F-]{36}}", vaultHandler.DeleteVault).Methods("DELETE")

	// Title-based memory access routes (registered after UUID routes, rely on UUID regex to disambiguate)
	router.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}", memoryHandler.GetMemoryByTitle).Methods("GET")
	router.HandleFunc("/api/users/{userId}/vaults/{vaultTitle}/memories", memoryHandler.ListMemoriesByVaultTitle).Methods("GET")

	return router
}
