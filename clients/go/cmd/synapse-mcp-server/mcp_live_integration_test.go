//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/synapse/synapse-mcp-server/client"
)

func TestMCPLiveEndToEnd(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	sdk := client.New(baseURL)

	// --- bootstrap user + memory via SDK ---
	email := fmt.Sprintf("mcp-it-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := sdk.CreateUser(context.Background(), client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	vault, err := sdk.CreateVault(context.Background(), user.ID, client.CreateVaultRequest{Title: "it-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mem, err := sdk.CreateMemoryInVault(context.Background(), user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "it-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer sdk.Close() // Ensure queues are drained before context is cancelled

	// put_context via SDK and ensure it's written
	if _, err := sdk.PutMemoryContext(ctx, user.ID, vault.VaultID, mem.ID, "integration ctx"); err != nil {
		t.Fatalf("put_context: %v", err)
	}
	if err := sdk.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after put_context: %v", err)
	}

	// Defensive check: verify context was written successfully
	ctxRes, err := sdk.GetLatestMemoryContext(ctx, user.ID, vault.VaultID, mem.ID)
	if err != nil || ctxRes == nil {
		t.Fatalf("verify context after put: %v", err)
	}
	if m, ok := ctxRes.Context.(map[string]interface{}); !ok || m["activeContext"] != "integration ctx" {
		t.Fatalf("context verification failed: %#v", ctxRes.Context)
	}

	// add_entry via SDK
	if _, err := sdk.AddEntryInVault(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{
		RawEntry: "searchable term",
		Summary:  "sum",
	}); err != nil {
		t.Fatalf("add_entry: %v", err)
	}
	if err := sdk.AwaitConsistency(ctx, mem.ID); err != nil {
		t.Fatalf("await consistency after add_entry: %v", err)
	}

	// search via SDK with retry
	var sr *client.SearchResponse
	deadline := time.Now().Add(30 * time.Second)
	for {
		res, err := sdk.Search(ctx, client.SearchRequest{
			UserID:   user.ID,
			MemoryID: mem.ID,
			Query:    "searchable term",
			TopK:     5,
		})
		if err == nil && res != nil && res.Count > 0 {
			sr = res
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("search_memories failed after retries: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
	if sr.Count == 0 {
		t.Fatalf("search returned zero results")
	}

	t.Logf("Test completed successfully - found %d search results", sr.Count)
}
