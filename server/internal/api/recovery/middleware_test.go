package recovery

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMiddlewarePanic verifies that a panic inside the handler results in 500.
func TestMiddlewarePanic(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	// ensure minimal body returned
	if body, _ := io.ReadAll(rr.Body); len(body) == 0 {
		t.Fatalf("expected response body")
	}
}

// TestMiddlewarePassThru verifies regular handler passes untouched.
func TestMiddlewarePassThru(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
