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

// add_entry_e2e_test exercises the full live flow against a running backend:
//  1. create user → vault → memory
//  2. set context snapshot
//  3. enqueue entry (context auto-attached)
//  4. list entries & verify count
//  5. cleanup (memory, vault, user)
//
// Run with: go test -tags=integration ./tests/integration -v
func TestAddEntryE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c := client.New(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer c.Close()

	// create user
	email := fmt.Sprintf("it-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// create vault & memory
	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "it-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mem, err := c.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "Demo", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// set context
	_, err = c.PutContext(ctx, user.ID, vault.VaultID, mem.ID, client.PutContextRequest{Context: map[string]interface{}{"activeContext": "ctx"}})
	if err != nil {
		t.Fatalf("put context: %v", err)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// add entry
	_, err = c.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: "hello", Summary: "sum"})
	if err != nil {
		t.Fatalf("add entry: %v", err)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)

	// list entries
	lr, err := c.ListEntries(ctx, user.ID, vault.VaultID, mem.ID, nil)
	if err != nil {
		t.Fatalf("list entries failed: %v", err)
	}
	if lr.Count == 0 {
		t.Fatalf("expected >0 entries, got 0")
	}

	// cleanup
	_ = c.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = c.DeleteUser(ctx, user.ID)
}
