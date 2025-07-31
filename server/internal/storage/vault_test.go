package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestListVaults_DecodeUUID ensures that ListVaults returns vaults without decoding errors.
func TestListVaults_DecodeUUID(t *testing.T) {
	ctx := context.Background()

	if spannerClient == nil {
		t.Fatal("spanner client not initialised – emulator setup failed")
	}

	// Clean slate before test
	if err := cleanupTables(ctx); err != nil {
		t.Fatalf("cleanup tables: %v", err)
	}

	storage := NewSpannerStorage(spannerClient)

	// Create a user – prerequisite for vault
	userReq := CreateUserRequest{Email: "vaulttest@example.com", TimeZone: "UTC"}
	user, err := storage.CreateUser(ctx, userReq)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a vault with a generated UUID
	vaultID := uuid.New()
	title := "My First Vault"

	_, err = storage.CreateVault(ctx, CreateVaultRequest{
		UserID:  user.UserID,
		VaultID: vaultID,
		Title:   title,
	})
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	// Call ListVaults
	vts, err := storage.ListVaults(ctx, user.UserID)
	if err != nil {
		t.Fatalf("list vaults: %v", err)
	}

	if len(vts) != 1 {
		t.Fatalf("expected 1 vault, got %d", len(vts))
	}

	v := vts[0]
	if v.VaultID != vaultID {
		t.Errorf("vaultID mismatch: expected %s, got %s", vaultID, v.VaultID)
	}
	if v.Title != title {
		t.Errorf("title mismatch: expected %s, got %s", title, v.Title)
	}
}
