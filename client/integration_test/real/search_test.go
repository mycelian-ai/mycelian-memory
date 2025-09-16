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

// TestSearchE2E performs comprehensive end-to-end search testing with step verification.
// Combines search flow testing with individual backend component validation.
func TestSearchE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// 1. User management is now external - use MockAuthorizer's actor ID

	// 2. create vault & memory and verify
	vaultTitle := fmt.Sprintf("search-vault-%s", uuid.NewString()[:8])
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil || vault.VaultID == "" {
		t.Fatalf("create vault failed: %v", err)
	}
	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{Title: "search-mem", MemoryType: "NOTES"})
	if err != nil || mem.ID == "" {
		t.Fatalf("create memory failed: %v", err)
	}

	// 3. write context and wait for consistency
	if _, err := c.PutContext(ctx, vault.VaultID, mem.ID, "integration context"); err != nil {
		t.Fatalf("put context: %v", err)
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after context: %v", err)
	}

	// 4. add keyword entries and verify via ListEntries
	for i := 0; i < 3; i++ {
		raw := fmt.Sprintf("the quick brown fox %d", i)
		if _, err := c.AddEntry(ctx, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: raw, Summary: "story"}); err != nil {
			t.Fatalf("add entry %d: %v", i, err)
		}
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after entries: %v", err)
	}

	// verify entries were added
	entries, err := c.ListEntries(ctx, vault.VaultID, mem.ID, nil)
	if err != nil || entries.Count != 3 {
		t.Fatalf("list entries unexpected: err=%v count=%d (expected 3)", err, entries.Count)
	}

	// small delay to ensure indexer processed entries
	time.Sleep(2 * time.Second)

	// 5. perform search with retry mechanism (handles indexer lag)
	var sr *client.SearchResponse
	deadline := time.Now().Add(20 * time.Second)
	for {
		topKE := 3
		topKC := 2
		sr, err = c.Search(ctx, client.SearchRequest{MemoryID: mem.ID, Query: "fox", TopKE: &topKE, TopKC: &topKC})
		if err == nil && sr.Count > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("search retries exhausted: last err %v, sr %#v", err, sr)
		}
		time.Sleep(1 * time.Second)
	}

	// 6. validate search results and context deeply
	if sr.Count == 0 {
		t.Fatalf("search returned zero results")
	}
	if string(sr.LatestContext) != "\"integration context\"" && string(sr.LatestContext) != "integration context" {
		t.Fatalf("latestContext mismatch: %s", string(sr.LatestContext))
	}
	if sr.LatestContextTimestamp == nil {
		t.Fatalf("latestContextTimestamp nil")
	}

	t.Logf("search completed successfully: found %d results with valid context", sr.Count)

	// cleanup
	_ = c.DeleteMemory(ctx, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, vault.VaultID)
	// User deletion is now external - no user cleanup needed
}
