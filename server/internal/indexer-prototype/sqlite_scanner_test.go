package indexer

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"memory-backend/internal/localstate"
	sqlstorage "memory-backend/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func setupTempSQLite(t *testing.T) (path string, db *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "mem.db")
	db, err := sqlstorage.Open(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := localstate.EnsureSQLiteSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return path, db
}

func TestSQLiteScanner_FetchEntriesSince(t *testing.T) {
	path, db := setupTempSQLite(t)
	defer db.Close()

	// seed data
	uid := uuid.New().String()
	vault := uuid.New().String()
	mid := uuid.New().String()
	base := time.Now().Add(-time.Hour)

	stmt := `INSERT INTO MemoryEntries (UserId, VaultId, MemoryId, Title, CreationTime, EntryId, RawEntry, Summary, Metadata, Tags) VALUES (?,?,?,?,?,?,?,?,?,?)`
	// older record (should be skipped)
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", base, uuid.New().String(), "old entry", "old", "{}", "{}")
	// newer records
	t1 := base.Add(10 * time.Second)
	t2 := base.Add(20 * time.Second)
	eid1 := uuid.New().String()
	eid2 := uuid.New().String()
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", t1, eid1, "hello", "sum", "{}", "{}")
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", t2, eid2, "world", "sum", "{}", "{}")

	logger := zerolog.Nop()
	sc, err := NewSQLiteScanner(path, logger)
	if err != nil {
		t.Fatalf("new scanner: %v", err)
	}
	defer sc.Close()

	entries, err := sc.FetchEntriesSince(context.Background(), base, 10)
	if err != nil {
		t.Fatalf("fetch entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].EntryID != eid1 || entries[1].EntryID != eid2 {
		t.Fatalf("order mismatch")
	}
}

func TestSQLiteScanner_FetchContextsSince(t *testing.T) {
	path, db := setupTempSQLite(t)
	defer db.Close()

	uid := uuid.New().String()
	vault := uuid.New().String()
	mid := uuid.New().String()

	base := time.Now().Add(-time.Hour)
	stmt := `INSERT INTO MemoryContexts (UserId, VaultId, MemoryId, Title, ContextId, Context, CreationTime) VALUES (?,?,?,?,?,?,?)`
	// old
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", uuid.New().String(), `{"foo":"bar"}`, base)
	// new
	c1 := uuid.New().String()
	c2 := uuid.New().String()
	t1 := base.Add(5 * time.Second)
	t2 := base.Add(15 * time.Second)
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", c1, `{"a":1}`, t1)
	_, _ = db.Exec(stmt, uid, vault, mid, "m1", c2, `{"b":2}`, t2)

	sc, err := NewSQLiteScanner(path, zerolog.Nop())
	if err != nil {
		t.Fatalf("scanner: %v", err)
	}
	defer sc.Close()

	ctxs, err := sc.FetchContextsSince(context.Background(), base, 10)
	if err != nil {
		t.Fatalf("fetch ctx: %v", err)
	}
	if len(ctxs) != 2 {
		t.Fatalf("want 2, got %d", len(ctxs))
	}
	if ctxs[0].ContextID != c1 || ctxs[1].ContextID != c2 {
		t.Fatalf("order mismatch")
	}
}
