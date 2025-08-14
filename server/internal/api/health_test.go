package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ensure NewHealthHandler constructs without args and CheckHealth responds
func TestHealthHandler_CheckHealth(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.CheckHealth(w, req)
	if code := w.Result().StatusCode; code != http.StatusOK && code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: %d", code)
	}
}
