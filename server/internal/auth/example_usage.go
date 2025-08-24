package auth

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mycelian/mycelian-memory/server/internal/config"
)

// ExampleUsage demonstrates how to set up simplified API key authorization
func ExampleUsage(cfg *config.Config) *mux.Router {
	// Create the appropriate authorizer based on configuration
	factory := NewAuthorizerFactory(cfg)
	authorizer := factory.CreateAuthorizer()

	// Set up router - no middleware needed!
	router := mux.NewRouter()

	// Example handler that does everything inline
	router.HandleFunc("/api/vaults", func(w http.ResponseWriter, r *http.Request) {
		// Extract API key directly from request
		apiKey, err := ExtractAPIKey(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Single call: validate API key + check vault.create permission
		actorInfo, err := authorizer.Authorize(r.Context(), apiKey, "vault.create", "default")
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Use actor info for vault operations
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Authorized actor: ` + actorInfo.ActorID + `"}`))
	}).Methods("POST")

	return router
}

// Configuration examples:
//
// SaaS Mode (Production):
// - Set MEMORY_SERVER_DEV_MODE=false (or leave unset)
// - Client provides API key via "Authorization: Bearer <api_key>" header
// - Handlers call ExtractAPIKey(r) then authorizer.Authorize()
// - ProductionAuthorizer validates against real auth provider
//
// Local Development Mode:
// - Set MEMORY_SERVER_DEV_MODE=true
// - Client uses hardcoded API key: "sk_local_mycelian_dev_key"
// - Handlers call ExtractAPIKey(r) then authorizer.Authorize()
// - MockAuthorizer resolves to "mycelian-dev" actor with admin permissions
//
// Handler Pattern (No Middleware):
// apiKey, err := auth.ExtractAPIKey(r)
// actorInfo, err := authorizer.Authorize(ctx, apiKey, "operation", "resource")
// // Use actorInfo.ActorID for business logic
