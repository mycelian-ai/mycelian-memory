package api

import (
	"github.com/gorilla/mux"
)

// NewRouter creates a new HTTP router with all API routes using clean architecture
func NewRouter() *mux.Router {
	router := mux.NewRouter()

	// Global middlewares
	router.Use(Recover)

	// Create handlers
	healthHandler := NewHealthHandler()
	router.HandleFunc("/api/health", healthHandler.CheckHealth).Methods("GET")

	// Legacy router trimmed to health only; routes wired in composition root.

	return router
}
