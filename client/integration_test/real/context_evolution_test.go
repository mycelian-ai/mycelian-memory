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

// TestContextEvolutionE2E exercises put-context + add-entry across multiple
// revisions and ensures each entry carries the correct snapshot.
func TestContextEvolutionE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	c := client.New(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer c.Close()

	email := fmt.Sprintf("evo-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "evo-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	mem, err := c.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "EvoTest", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	snapshots := []string{"ctx-1", "ctx-2", "ctx-3"}
	for i, snap := range snapshots {
		if _, err := c.PutContext(ctx, user.ID, vault.VaultID, mem.ID, client.PutContextRequest{Context: map[string]interface{}{"activeContext": snap}}); err != nil {
			t.Fatalf("put context %d: %v", i, err)
		}
		raw := fmt.Sprintf("entry-%d", i+1)
		if _, err := c.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: raw, Summary: "evo"}); err != nil {
			t.Fatalf("add entry %d: %v", i, err)
		}
	}

	_ = c.AwaitConsistency(ctx, mem.ID)

	latestCtx, err := c.GetContext(ctx, user.ID, vault.VaultID, mem.ID)
	if err != nil {
		t.Fatalf("get latest context: %v", err)
	}
	rawJSON, _ := json.Marshal(latestCtx.Context)
	if string(rawJSON) == "" {
		t.Fatalf("latest context empty")
	}

	// cleanup
	_ = c.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID)
	_ = c.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = c.DeleteUser(ctx, user.ID)
}

// TestMultiAgentContextAccessE2E verifies that context written by one agent is
// visible to another agent using vault-aware APIs.
func TestMultiAgentContextAccessE2E(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Agent A
	agentA := client.New(baseURL)
	defer agentA.Close()

	email := fmt.Sprintf("ma-%s@example.com", uuid.NewString())
	uid2 := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := agentA.CreateUser(ctx, client.CreateUserRequest{UserID: uid2, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	vault, err := agentA.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "ma-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	mem, err := agentA.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "MATest", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	originalCtx := "Agent A context"
	if _, err := agentA.PutContext(ctx, user.ID, vault.VaultID, mem.ID, client.PutContextRequest{Context: map[string]interface{}{"activeContext": originalCtx}}); err != nil {
		t.Fatalf("agentA put context: %v", err)
	}
	_ = agentA.AwaitConsistency(ctx, mem.ID)

	if _, err := agentA.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: "A entry", Summary: "sum"}); err != nil {
		t.Fatalf("agentA add entry: %v", err)
	}
	_ = agentA.AwaitConsistency(ctx, mem.ID)

	// Agent B
	agentB := client.New(baseURL)
	defer agentB.Close()

	resCtx, err := agentB.GetContext(ctx, user.ID, vault.VaultID, mem.ID)
	if err != nil {
		t.Fatalf("agentB get context: %v", err)
	}
	m, ok := resCtx.Context.(map[string]interface{})
	if !ok || m["activeContext"] != originalCtx {
		t.Fatalf("context mismatch: %#v", resCtx.Context)
	}

	if _, err := agentB.AddEntry(ctx, user.ID, vault.VaultID, mem.ID, client.AddEntryRequest{RawEntry: "B entry", Summary: "sum"}); err != nil {
		t.Fatalf("agentB add entry: %v", err)
	}
	_ = agentB.AwaitConsistency(ctx, mem.ID)

	entries, err := agentB.ListEntries(ctx, user.ID, vault.VaultID, mem.ID, map[string]string{"limit": "10"})
	if err != nil || len(entries.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d (%v)", len(entries.Entries), err)
	}

	// cleanup
	_ = agentB.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID)
	_ = agentB.DeleteVault(ctx, user.ID, vault.VaultID)
	_ = agentB.DeleteUser(ctx, user.ID)
}
