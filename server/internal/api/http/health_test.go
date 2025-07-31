package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memory-backend/internal/storage/sqlite"
)

func TestCheckStorageHealth_SQLite(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/health/db", nil)
	w := httptest.NewRecorder()
	h.CheckStorageHealth(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Result().StatusCode)
	}
}
