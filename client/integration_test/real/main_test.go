//go:build integration
// +build integration

package client_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestMain waits for the memory-service health endpoint before running tests.
func TestMain(m *testing.M) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	waitForHealthy(baseURL, 30*time.Second)
	os.Exit(m.Run())
}

func waitForHealthy(baseURL string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			var body struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && body.Status == "UP" {
				_ = resp.Body.Close()
				return
			}
			_ = resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	// If not healthy within timeout, fail fast
	panic("memory-service not healthy at /api/health within timeout")
}
