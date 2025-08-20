//go:build integration
// +build integration

package client_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/client"
)

// TestUserVaultMemoryCRUD covers end-to-end CRUD for user, vault, and memory.
func TestUserVaultMemoryCRUD(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11545"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.NewWithDevMode(baseURL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	defer c.Close()

	// User management is now external - use MockAuthorizer's actor ID

	// create vault
	vault, err := c.CreateVault(ctx, client.CreateVaultRequest{Title: "crud-vault"})
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if vault.VaultID == "" {
		t.Fatal("CreateVault: empty vault ID")
	}

	// list vaults
	vaults, err := c.ListVaults(ctx)
	if err != nil {
		t.Fatalf("ListVaults: %v", err)
	}
	if len(vaults) == 0 {
		t.Fatal("ListVaults: expected at least one vault")
	}

	// get vault
	gotVault, err := c.GetVault(ctx, vault.VaultID)
	if err != nil {
		t.Fatalf("GetVault: %v", err)
	}
	if gotVault.VaultID != vault.VaultID {
		t.Fatalf("GetVault: id mismatch: %s != %s", gotVault.VaultID, vault.VaultID)
	}

	// create memory
	mem, err := c.CreateMemory(ctx, vault.VaultID, client.CreateMemoryRequest{Title: "crud-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	if mem.ID == "" {
		t.Fatal("CreateMemory: empty ID")
	}

	// list memories
	mems, err := c.ListMemories(ctx, vault.VaultID)
	if err != nil {
		t.Fatalf("ListMemories: %v", err)
	}
	if len(mems) == 0 {
		t.Fatal("ListMemories: expected at least one memory")
	}

	// get memory
	gotMem, err := c.GetMemory(ctx, vault.VaultID, mem.ID)
	if err != nil {
		t.Fatalf("GetMemory: %v", err)
	}
	if gotMem.ID != mem.ID {
		t.Fatalf("GetMemory: id mismatch: %s != %s", gotMem.ID, mem.ID)
	}

	// cleanup
	if err := c.DeleteMemory(ctx, vault.VaultID, mem.ID); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if err := c.DeleteVault(ctx, vault.VaultID); err != nil {
		t.Fatalf("DeleteVault: %v", err)
	}
	// User deletion is now external - no user cleanup needed
}
