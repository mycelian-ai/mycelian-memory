//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// env returns the value of key or the provided fallback when the env var is unset.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ping checks that a GET request to the given URL returns HTTP 200.
// It is used to quickly skip tests when the dev stack is not running.
func ping(url string) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", r.StatusCode)
	}
	return nil
}

// mustJSON decodes the HTTP response body into v or fails the test with context.
func mustJSON(t *testing.T, resp *http.Response, v interface{}) {
	if resp == nil {
		t.Fatalf("nil response")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("http %d: %s", resp.StatusCode, string(body))
	}
	// Strict decode; call sites handle any schema variation
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

// waitForHealthy polls /v0/health until the memory-service responds HTTP 200
// with a JSON body containing a non-empty status field ("healthy" or "unhealthy"),
// or the timeout elapses. This endpoint is non-blocking on live checks.
func waitForHealthy(t *testing.T, baseURL string, timeout time.Duration) {
	t.Logf("Checking memory-service health at %s/v0/health (timeout %s)", baseURL, timeout)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/v0/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			var data struct {
				Status string `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.Status != "" {
				_ = resp.Body.Close()
				return // responding
			}
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("memory-service /v0/health not responding within %s", timeout)
}

func mustNewRequest(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}

// ensureUser returns the userId to be used by tests.
// With external user identification, we simply return the provided userID
// as user management is now handled externally.
func ensureUser(t *testing.T, memSvc, userID, email string) string {
	// User management is now external - simply return the userID for use in tests
	// In development mode, the server uses the configured DevUserID from environment
	return userID
}
