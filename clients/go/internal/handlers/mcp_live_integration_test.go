//go:build integration
// +build integration

package handlers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/synapse/synapse-mcp-server/client"
)

// This test exercises the real backend via all MCP handlers in-process.
func TestMCPHandlersLiveEndToEnd(t *testing.T) {
	base := os.Getenv("TEST_BACKEND_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	sdk := client.New(base)

	ctx := context.Background()

	// bootstrap user + memory via SDK helpers (simpler than tool path)
	email := fmt.Sprintf("live-%s@example.com", uuid.NewString())
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	user, err := sdk.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	vault, err := sdk.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "live-vault"})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	mem, err := sdk.CreateMemoryInVault(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "live-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	// Handlers
	ch := NewContextHandler(sdk)
	eh := NewEntryHandler(sdk)
	cons := NewConsistencyHandler(sdk)
	sh := NewSearchHandler(sdk)

	// put_context via handler
	if _, err := ch.handlePutContext(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   user.ID,
		"vault_id":  vault.VaultID,
		"memory_id": mem.ID,
		"content":   "ctx-live",
	}}}); err != nil {
		t.Fatalf("put_context: %v", err)
	}

	// await consistency
	if _, err := cons.handleAwait(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"vault_id":  vault.VaultID,
		"memory_id": mem.ID,
	}}}); err != nil {
		t.Fatalf("await: %v", err)
	}

	// add_entry
	if _, err := eh.handleAddEntry(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"user_id":   user.ID,
		"vault_id":  vault.VaultID,
		"memory_id": mem.ID,
		"raw_entry": "live search term",
		"summary":   "sum",
	}}}); err != nil {
		t.Fatalf("add_entry: %v", err)
	}
	_ = sdk.AwaitConsistency(ctx, mem.ID)

	// retry search until found
	deadline := time.Now().Add(30 * time.Second)
	for {
		res, err := sh.handleSearch(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
			"user_id":   user.ID,
			"vault_id":  vault.VaultID,
			"memory_id": mem.ID,
			"query":     "live search term",
		}}})
		if err == nil && res != nil && !res.IsError {
			t.Logf("search ok: %s", res.Content[0])
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("search did not succeed: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
}
