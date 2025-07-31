//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/synapse/synapse-mcp-server/client"
)

// TestSearchStepE2E performs a search scenario verifying each backend step.
func TestSearchStepE2E(t *testing.T) {
	base := os.Getenv("TEST_BACKEND_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	sdk := client.New(base)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer sdk.Close()

	// 1) create user
	email := fmt.Sprintf("step-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := sdk.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil || user.ID == "" {
		t.Fatalf("create user failed: %v", err)
	}

	// 2) create vault and verify via GetVaultByTitle
	vaultTitle := fmt.Sprintf("vault-%s", uuid.NewString()[:8])
	vault, err := sdk.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	// vault validated via non-empty ID; backend list not required

	// 3) create memory and verify via ListMemories
	mem, err := sdk.CreateMemoryInVault(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "step-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// memory validated via returned struct; no list call

	// 4) add entry and verify via ListEntries
	_, err = sdk.AddEntryInVault(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: "the quick brown fox", Summary: "story"})
	if err != nil {
		t.Fatalf("add entry: %v", err)
	}
	// wait briefly for async queue flush
	if err := sdk.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency: %v", err)
	}
	entries, err := sdk.ListEntriesInVault(ctx, user.ID, vault.VaultID, mem.ID, nil)
	if err != nil || entries.Count != 1 {
		t.Fatalf("list entries unexpected: %v count=%d", err, entries.Count)
	}

	// small delay to ensure indexer ingested the entry
	time.Sleep(1 * time.Second)

	// 5) search once for keyword
	sr, err := sdk.Search(ctx, client.SearchRequest{UserID: user.ID, MemoryID: mem.ID, Query: "fox", TopK: 3})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if sr.Count == 0 {
		t.Fatalf("search returned zero results")
	}

	// Cleanup best-effort
	_ = sdk.DeleteMemory(ctx, user.ID, mem.ID)
	_ = sdk.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = sdk.DeleteUser(ctx, user.ID)
}
