package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Deprecated: storage health endpoint removed; keep a minimal test to ensure handler compiles
func TestCheckStorageHealth_Deprecated(t *testing.T) {
	h := NewHealthHandler(nil)

	// Call the handler directly without asserting route exposure
	req := httptest.NewRequest(http.MethodGet, "/api/health/db", nil)
	w := httptest.NewRecorder()
	h.CheckStorageHealth(w, req)

	// Accept either 200 or 503 response; routing no longer exposes this endpoint
	if code := w.Result().StatusCode; code != http.StatusOK && code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", code)
	}
}
