//go:build local
// +build local

package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/mycelian/mycelian-memory/server/internal/localstate"
	"github.com/mycelian/mycelian-memory/server/internal/storage"

	"github.com/google/uuid"
)

// setupTempSQLite creates a temporary on-disk SQLite database with schema applied.
func setupTempSQLite(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mem.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := localstate.EnsureSQLiteSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return path, db
}

func newTestAdapter(t *testing.T) storage.Storage {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	if err := localstate.EnsureSQLiteSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	adap, err := NewSqliteStorageWithDB(db)
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	return adap
}

func TestUserMemoryCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestAdapter(t)

	// Create user
	u, err := s.CreateUser(ctx, storage.CreateUserRequest{Email: "test@example.com", TimeZone: "UTC"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Get by email
	got, err := s.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if got.UserID != u.UserID {
		t.Fatalf("expected same user id, got %s want %s", got.UserID, u.UserID)
	}

	// Create vault
	vid := uuid.New()
	// Memory creation now requires VaultID
	mem, err := s.CreateMemory(ctx, storage.CreateMemoryRequest{UserID: u.UserID, VaultID: vid, MemoryType: "CONVERSATION", Title: "Test", Description: nil})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// List memories
	list, err := s.ListMemories(ctx, u.UserID, vid)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(list) != 1 || list[0].MemoryID != mem.MemoryID {
		t.Fatalf("unexpected list result: %+v", list)
	}

	// Create entry
	entryReq := storage.CreateMemoryEntryRequest{
		UserID:   u.UserID,
		VaultID:  vid,
		MemoryID: mem.MemoryID,
		RawEntry: "hello",
	}
	entry, err := s.CreateMemoryEntry(ctx, entryReq)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	// List entries
	ents, err := s.ListMemoryEntries(ctx, storage.ListMemoryEntriesRequest{UserID: u.UserID, VaultID: vid, MemoryID: mem.MemoryID})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(ents) != 1 || ents[0].EntryID != entry.EntryID {
		t.Fatalf("unexpected entries: %+v", ents)
	}

	// Soft delete entry
	if err := s.SoftDeleteMemoryEntry(ctx, u.UserID, vid, mem.MemoryID, entry.CreationTime); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	ents2, err := s.ListMemoryEntries(ctx, storage.ListMemoryEntriesRequest{UserID: u.UserID, VaultID: vid, MemoryID: mem.MemoryID})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(ents2) != 1 { // still returns but flagged deletion time; our list doesn't filter now, so just ensure no error
		t.Fatalf("unexpected list after delete: %v", ents2)
	}
}

func TestSqlite_GetVaultAndMemoryByTitle(t *testing.T) {
	_, db := setupTempSQLite(t)
	defer db.Close()

	store, err := NewSqliteStorageWithDB(db)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	userID := uuid.New().String()
	vaultID := uuid.New()
	vaultTitle := "vault-x"

	// insert vault
	_, err = db.Exec(`INSERT INTO Vaults (UserId, VaultId, Title, CreationTime) VALUES (?,?,?,datetime('now'))`, userID, vaultID.String(), vaultTitle)
	if err != nil {
		t.Fatalf("insert vault: %v", err)
	}

	// insert memory
	memID := uuid.New().String()
	memTitle := "mem-x"
	_, err = db.Exec(`INSERT INTO Memories (UserId, VaultId, MemoryId, MemoryType, Title, CreationTime) VALUES (?,?,?,?,?,datetime('now'))`, userID, vaultID.String(), memID, "PROJECT", memTitle)
	if err != nil {
		t.Fatalf("insert memory: %v", err)
	}

	// --- vault success ---
	v, err := store.GetVaultByTitle(context.Background(), userID, vaultTitle)
	if err != nil {
		t.Fatalf("get vault: %v", err)
	}
	if v.VaultID != vaultID {
		t.Fatalf("vault id mismatch")
	}

	// --- vault not found ---
	if _, err := store.GetVaultByTitle(context.Background(), userID, "nope"); err == nil {
		t.Fatalf("expected error for missing vault title")
	}

	// --- memory success ---
	m, err := store.GetMemoryByTitle(context.Background(), userID, vaultID, memTitle)
	if err != nil {
		t.Fatalf("get memory: %v", err)
	}
	if m.MemoryID != memID {
		t.Fatalf("memory id mismatch")
	}

	// --- memory not found ---
	if _, err := store.GetMemoryByTitle(context.Background(), userID, vaultID, "zzz"); err == nil {
		t.Fatalf("expected error for missing memory title")
	}
}
