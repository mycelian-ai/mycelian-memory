package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mycelian/mycelian-memory/server/internal/storage/sqlite"
)

// Deprecated: storage health endpoint removed; keep a minimal test to ensure handler compiles
func TestCheckStorageHealth_Deprecated(t *testing.T) {
	tmp, err := os.CreateTemp("", "memdb-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	defer os.Remove(tmp.Name())

	db, err := sqlite.Open(tmp.Name())
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	store, _ := sqlite.NewSqliteStorageWithDB(db)

	h := NewHealthHandler(store)

	// Call the handler directly without asserting route exposure
	req := httptest.NewRequest(http.MethodGet, "/api/health/db", nil)
	w := httptest.NewRecorder()
	h.CheckStorageHealth(w, req)

	// Accept either 200 or 503 response; routing no longer exposes this endpoint
	if code := w.Result().StatusCode; code != http.StatusOK && code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", code)
	}
}
