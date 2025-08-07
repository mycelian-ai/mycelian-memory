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

// TestUserVaultMemoryCRUD covers end-to-end CRUD for user, vault, and memory.
func TestUserVaultMemoryCRUD(t *testing.T) {
	baseURL := os.Getenv("TEST_BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := client.New(baseURL)
	defer c.Close()

	// create user
	uid := fmt.Sprintf("u%s", uuid.NewString()[:8])
	email := fmt.Sprintf("crud-%s@example.com", uuid.NewString())
	user, err := c.CreateUser(ctx, client.CreateUserRequest{UserID: uid, Email: email})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == "" {
		t.Fatal("CreateUser: empty user ID")
	}

	// get user
	fetched, err := c.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if fetched.ID != user.ID {
		t.Fatalf("GetUser: id mismatch: %s != %s", fetched.ID, user.ID)
	}

	// create vault
	vault, err := c.CreateVault(ctx, user.ID, client.CreateVaultRequest{Title: "crud-vault"})
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if vault.VaultID == "" {
		t.Fatal("CreateVault: empty vault ID")
	}

	// list vaults
	vaults, err := c.ListVaults(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListVaults: %v", err)
	}
	if len(vaults) == 0 {
		t.Fatal("ListVaults: expected at least one vault")
	}

	// get vault
	gotVault, err := c.GetVault(ctx, user.ID, vault.VaultID)
	if err != nil {
		t.Fatalf("GetVault: %v", err)
	}
	if gotVault.VaultID != vault.VaultID {
		t.Fatalf("GetVault: id mismatch: %s != %s", gotVault.VaultID, vault.VaultID)
	}

	// create memory
	mem, err := c.CreateMemory(ctx, user.ID, vault.VaultID, client.CreateMemoryRequest{Title: "crud-mem", MemoryType: "NOTES"})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	if mem.ID == "" {
		t.Fatal("CreateMemory: empty ID")
	}

	// list memories
	mems, err := c.ListMemories(ctx, user.ID, vault.VaultID)
	if err != nil {
		t.Fatalf("ListMemories: %v", err)
	}
	if len(mems) == 0 {
		t.Fatal("ListMemories: expected at least one memory")
	}

	// get memory
	gotMem, err := c.GetMemory(ctx, user.ID, vault.VaultID, mem.ID)
	if err != nil {
		t.Fatalf("GetMemory: %v", err)
	}
	if gotMem.ID != mem.ID {
		t.Fatalf("GetMemory: id mismatch: %s != %s", gotMem.ID, mem.ID)
	}

	// cleanup
	if err := c.DeleteMemory(ctx, user.ID, vault.VaultID, mem.ID); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if err := c.DeleteVault(ctx, user.ID, vault.VaultID); err != nil {
		t.Fatalf("DeleteVault: %v", err)
	}
	// Note: DELETE /api/users/{userId} is not currently supported by the backend.
	// Leave user as-is; cleanup focuses on memory and vault.
}
