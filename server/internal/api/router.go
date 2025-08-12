package api

import (
	httpHandlers "github.com/mycelian/mycelian-memory/server/internal/api/http"
	"github.com/mycelian/mycelian-memory/server/internal/api/recovery"

	"github.com/gorilla/mux"
)

// NewRouter creates a new HTTP router with all API routes using clean architecture
func NewRouter() *mux.Router {
	router := mux.NewRouter()

	// Global middlewares
	router.Use(recovery.Middleware)

	// Create handlers
	healthHandler := httpHandlers.NewHealthHandler()
	router.HandleFunc("/api/health", healthHandler.CheckHealth).Methods("GET")

	// Legacy router trimmed to health only; v2 routes wired in composition root.

	return router
}
