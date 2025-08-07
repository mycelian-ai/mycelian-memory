//go:build integration
// +build integration

package client_test

import (
	"context"
	"encoding/json"
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
		baseURL = "http://localhost:8080"
	}

	c := client.New(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// 1. create user and verify
	email := fmt.Sprintf("search-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil || user.ID == "" {
		t.Fatalf("create user failed: %v", err)
	}

	// 2. create vault & memory and verify
	vaultTitle := fmt.Sprintf("search-vault-%s", uuid.NewString()[:8])
	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil || vault.VaultID == "" {
		t.Fatalf("create vault failed: %v", err)
	}
	mem, err := c.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "search-mem", MemoryType: "NOTES"})
	if err != nil || mem.ID == "" {
		t.Fatalf("create memory failed: %v", err)
	}

	// 3. write context and wait for consistency
	_, err = c.PutContext(ctx, user.ID, vault.VaultID, mem.ID, client.PutContextRequest{Context: map[string]interface{}{"activeContext": "integration context"}})
	if err != nil {
		t.Fatalf("put context: %v", err)
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after context: %v", err)
	}

	// 4. add keyword entries and verify via ListEntries
	for i := 0; i < 3; i++ {
		raw := fmt.Sprintf("the quick brown fox %d", i)
		if _, err := c.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: raw, Summary: "story"}); err != nil {
			t.Fatalf("add entry %d: %v", i, err)
		}
	}
	if err := c.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after entries: %v", err)
	}

	// verify entries were added
	entries, err := c.ListEntries(ctx, user.ID, vault.VaultID, mem.ID, nil)
	if err != nil || entries.Count != 3 {
		t.Fatalf("list entries unexpected: err=%v count=%d (expected 3)", err, entries.Count)
	}

	// small delay to ensure indexer processed entries
	time.Sleep(2 * time.Second)

	// 5. perform search with retry mechanism (handles indexer lag)
	var sr *client.SearchResponse
	deadline := time.Now().Add(20 * time.Second)
	for {
		sr, err = c.Search(ctx, client.SearchRequest{UserID: user.ID, MemoryID: mem.ID, Query: "fox", TopK: 3})
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

	expectedCtx := "integration context"
	var val any
	if err := json.Unmarshal(sr.LatestContext, &val); err != nil {
		t.Fatalf("unmarshal latestContext: %v", err)
	}
	switch v := val.(type) {
	case string:
		var raw2 string
		if err := json.Unmarshal([]byte(v), &raw2); err == nil {
			v = raw2
		}
		if v != expectedCtx {
			var frag map[string]string
			if err := json.Unmarshal([]byte(v), &frag); err != nil || frag["activeContext"] != expectedCtx {
				t.Fatalf("latestContext mismatch: %q", v)
			}
		}
	case map[string]interface{}:
		if s, ok := v["activeContext"].(string); !ok || s != expectedCtx {
			t.Fatalf("latestContext mismatch: %#v", v)
		}
	default:
		t.Fatalf("unexpected latestContext type: %#v", v)
	}
	if sr.ContextTimestamp == nil {
		t.Fatalf("contextTimestamp nil")
	}

	t.Logf("search completed successfully: found %d results with valid context", sr.Count)

	// cleanup
	_ = c.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = c.DeleteUser(ctx, user.ID)
}
