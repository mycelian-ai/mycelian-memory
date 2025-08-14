//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
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

// waitForHealthy polls /api/health until the memory-service responds HTTP 200
// with a JSON body containing a non-empty status field ("healthy" or "unhealthy"),
// or the timeout elapses. This endpoint is non-blocking on live checks.
func waitForHealthy(t *testing.T, baseURL string, timeout time.Duration) {
	t.Logf("Checking memory-service health at %s/api/health (timeout %s)", baseURL, timeout)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
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
	t.Fatalf("memory-service /api/health not responding within %s", timeout)
}

// ensureWeaviateTenants adds the given tenant to both MemoryEntry and MemoryContext classes.
func ensureWeaviateTenants(t *testing.T, weaviateURL, tenant string) {
	// Fallback approach: trigger tenant creation implicitly by creating a dummy object then deleting it
	// This works in older Weaviate without explicit tenant endpoints
	for _, class := range []string{"MemoryEntry", "MemoryContext"} {
		// Minimal payload with required fields
		id := "00000000-0000-0000-0000-000000000000"
		payload := fmt.Sprintf(`{"userId":%q}`, tenant)
		// Create
		url := fmt.Sprintf("%s/v1/objects", weaviateURL)
		body := fmt.Sprintf(`{"class":"%s","id":%q,"tenant":%q,"properties":%s}`, class, id, tenant, payload)
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
		// Delete best-effort
		delURL := fmt.Sprintf("%s/v1/objects/%s/%s?tenant=%s", weaviateURL, class, id, tenant)
		_, _ = http.DefaultClient.Do(mustNewRequest(http.MethodDelete, delURL))
	}
}

func mustNewRequest(method, url string) *http.Request {
	req, _ := http.NewRequest(method, url, nil)
	return req
}

// ensureUser makes sure a specific user exists; creates it if missing.
// Returns the userId to be used by tests.
func ensureUser(t *testing.T, memSvc, userID, email string) string {
	// Fast path: exists
	r, err := http.Get(fmt.Sprintf("%s/api/users/%s", memSvc, userID))
	if err == nil && r.StatusCode == http.StatusOK {
		_ = r.Body.Close()
		return userID
	}
	if r != nil {
		_ = r.Body.Close()
	}
	// Create user
	payload := fmt.Sprintf(`{"userId":"%s","email":"%s","timeZone":"UTC","displayName":"Test User"}`, userID, email)
	resp, err := http.Post(memSvc+"/api/users", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("create user %s: %v", userID, err)
	}
	// If created, decode; otherwise tolerate duplicate by verifying via GET
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var out struct {
			UserID string `json:"userId"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode user create: %v", err)
		}
		_ = resp.Body.Close()
		return out.UserID
	}
	// Fallback: check again
	_ = resp.Body.Close()
	r2, err := http.Get(fmt.Sprintf("%s/api/users/%s", memSvc, userID))
	if err == nil && r2.StatusCode == http.StatusOK {
		_ = r2.Body.Close()
		return userID
	}
	if r2 != nil {
		_ = r2.Body.Close()
	}
	t.Fatalf("failed to ensure user %s: status %d", userID, resp.StatusCode)
	return ""
}
