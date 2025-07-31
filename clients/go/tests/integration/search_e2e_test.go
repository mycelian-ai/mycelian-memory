//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/synapse/synapse-mcp-server/client"
)

func TestSearchE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c := client.New(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	// 1. create user
	email := fmt.Sprintf("search-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 2. create vault & memory
	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "search-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mem, err := c.CreateMemoryInVault(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "search-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// 3. write context and wait
	_, err = c.PutMemoryContext(ctx, user.ID, vault.VaultID, mem.ID, "integration context")
	if err != nil {
		t.Fatalf("put context: %v", err)
	}
	_ = c.AwaitConsistency(ctx, mem.ID)
	time.Sleep(2 * time.Second)

	// 4. add keyword entries
	for i := 0; i < 3; i++ {
		raw := fmt.Sprintf("the quick brown fox %d", i)
		if _, err := c.AddEntryInVault(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: raw, Summary: "story"}); err != nil {
			t.Fatalf("add entry: %v", err)
		}
	}
	_ = c.AwaitConsistency(ctx, mem.ID)
	time.Sleep(2 * time.Second)

	// 5. perform search with retry (tenant setup)
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

	// cleanup
	_ = c.DeleteMemory(ctx, user.ID, mem.ID)
	_ = c.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = c.DeleteUser(ctx, user.ID)
}
