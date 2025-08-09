//go:build integration
// +build integration

package client_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mycelian/mycelian-memory/client"
)

// TestDeleteEntryPropagationE2E verifies that deleting an entry removes it from search results.
func TestDeleteEntryPropagationE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c := client.New(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// 1) Create user, vault, memory
	email := fmt.Sprintf("del-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "del-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mem, err := c.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "del-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// 2) Add entry with unique content, wait for indexer to upsert
	uniqueText := fmt.Sprintf("unique-delete-token-%s", uuid.NewString())
	if _, err := c.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: uniqueText, Summary: "to be deleted"}); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after add: %v", err)
	}

	// 3) Poll search until the entry appears
	var appeared bool
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		sr, _ := c.Search(ctx, client.SearchRequest{UserID: user.ID, MemoryID: mem.ID, Query: uniqueText, TopK: 3})
		if sr != nil && sr.Count > 0 {
			appeared = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !appeared {
		t.Fatalf("entry never appeared in search before delete")
	}

	// 4) Find entryId via ListEntries, then delete it
	lr, err := c.ListEntries(ctx, user.ID, vault.VaultID, mem.ID, nil)
	if err != nil || lr.Count == 0 {
		t.Fatalf("list entries: err=%v count=%d", err, lr.Count)
	}
	entryID := lr.Entries[0].ID
	if entryID == "" {
		t.Fatalf("empty entryID")
	}
	if err := c.DeleteEntry(ctx, user.ID, vault.VaultID, mem.ID, entryID); err != nil {
		t.Fatalf("delete entry: %v", err)
	}

	// 5) Poll search until the entry disappears
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		sr, _ := c.Search(ctx, client.SearchRequest{UserID: user.ID, MemoryID: mem.ID, Query: uniqueText, TopK: 3})
		if sr != nil && sr.Count == 0 {
			break // success
		}
		time.Sleep(500 * time.Millisecond)
	}

	sr, _ := c.Search(ctx, client.SearchRequest{UserID: user.ID, MemoryID: mem.ID, Query: uniqueText, TopK: 3})
	if sr != nil && sr.Count > 0 {
		t.Fatalf("entry still present in search after delete: %+v", sr)
	}

	// Cleanup resources
	_ = c.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = c.DeleteUser(ctx, user.ID)
}
