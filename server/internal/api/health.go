package api

import (
	"net/http"
	"sync/atomic"
	"time"

	respond "github.com/mycelian/mycelian-memory/server/internal/api/respond"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// global health flag (1 = healthy, 0 = unhealthy)
var healthyFlag atomic.Int32

// lastProbeErr keeps the most recent dependency failure details.
var lastProbeErr atomic.Value // string (deprecated)

func init() {
	healthyFlag.Store(0)
	lastProbeErr.Store("")
}

// BindServiceHealth allows run.go to inject the service health function.
var serviceIsHealthy func() bool = func() bool { return healthyFlag.Load() == 1 }

func BindServiceHealth(f func() bool) { serviceIsHealthy = f }

// CheckHealth handles GET /api/health
// Always returns 200; body reports healthy/unhealthy. 500 indicates handler failure only.
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	status := "unhealthy"
	if serviceIsHealthy() {
		status = "healthy"
	}
	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	respond.WriteJSON(w, http.StatusOK, response)
}
