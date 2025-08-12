package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	platformHttp "github.com/mycelian/mycelian-memory/server/internal/platform/http"
)

// HealthHandler handles health check endpoints
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// global health flag (1 = healthy, 0 = unhealthy)
var healthyFlag atomic.Int32

// lastProbeErr keeps the most recent dependency failure details.
var lastProbeErr atomic.Value // string

func init() {
	healthyFlag.Store(1)
	lastProbeErr.Store("")
}

// StartHealthMonitor launches a background goroutine that probes Waviate vector store
// and (if provider==ollama) Ollama model every `interval`.
// vectorStore must be "waviate" (others ignored).
func StartHealthMonitor(ctx context.Context, waviateURL, embedProvider, embedModel string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		probe := func() {
			// Vector store check (only waviate supported)
			var errVector error
			if waviateURL != "" {
				errVector = checkWaviateEndpoint(waviateURL)
			}

			// Ollama check if required
			var errOllama error
			if strings.EqualFold(embedProvider, "ollama") {
				errOllama = checkOllamaModel(embedModel)
			}

			// Collect error details
			errs := []string{}
			if errVector != nil {
				errs = append(errs, fmt.Sprintf("waviate %s: %v", waviateURL, errVector))
			}
			if errOllama != nil {
				base := os.Getenv("OLLAMA_URL")
				if base == "" {
					base = "http://localhost:11434"
				}
				errs = append(errs, fmt.Sprintf("ollama %s: %v", base, errOllama))
			}

			if len(errs) == 0 {
				healthyFlag.Store(1)
				lastProbeErr.Store("")
			} else {
				healthyFlag.Store(0)
				lastProbeErr.Store(strings.Join(errs, "; "))
			}
		}

		// initial probe immediately
		probe()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				probe()
			}
		}
	}()
}

// checkWaviateEndpoint returns nil if GET http://<host>/v1/meta succeeds with 200.
func checkWaviateEndpoint(base string) error {
	if base == "" {
		return fmt.Errorf("waviate URL missing")
	}
	url := base
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		url = "http://" + base
	}
	resp, err := http.Get(url + "/v1/meta")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("waviate status %d", resp.StatusCode)
	}
	return nil
}

// checkOllamaModel verifies that the given model appears in /api/tags.
func checkOllamaModel(model string) error {
	base := os.Getenv("OLLAMA_URL")
	if base == "" {
		base = "http://localhost:11434"
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	resp, err := http.Get(base + "/api/tags")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}
	target := strings.Split(model, ":")[0]
	for _, m := range data.Models {
		name := strings.Split(m.Name, ":")[0]
		if name == target {
			return nil
		}
	}
	return fmt.Errorf("model %s not found in tag list", model)
}

// CheckHealth handles GET /api/health
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	if healthyFlag.Load() == 1 {
		response := map[string]interface{}{
			"status":    "UP",
			"message":   "Service is healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		platformHttp.WriteJSON(w, http.StatusOK, response)
		return
	}

	errMsg, _ := lastProbeErr.Load().(string)
	if errMsg == "" {
		errMsg = "One or more dependencies unavailable"
	}
	response := map[string]interface{}{
		"status":    "DOWN",
		"message":   errMsg,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	platformHttp.WriteJSON(w, http.StatusInternalServerError, response)
}

// Deprecated storage health route removed.
