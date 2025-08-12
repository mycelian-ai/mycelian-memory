package postgres

import (
	"os"
	"testing"

	"github.com/mycelian/mycelian-memory/server/internal/store"
	"github.com/mycelian/mycelian-memory/server/internal/store/storetest"
)

func makePGStore(t *testing.T) store.Store {
	t.Helper()
	dsn := os.Getenv("MEMORY_BACKEND_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("MEMORY_BACKEND_POSTGRES_DSN not set; skipping postgres store integration test")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatalf("postgres open: %v", err)
	}
	return NewWithDB(db)
}

func TestPostgresStore_Compliance(t *testing.T) {
	storetest.Run(t, makePGStore)
}
