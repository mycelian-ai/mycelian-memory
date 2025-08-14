package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_VaultCRUD_AndGetByTitle(t *testing.T) {
	t.Parallel()

	userID := "user123"
	vaultID := "vault-456"
	vaultTitle := "work-projects"

	v := client.Vault{UserID: userID, VaultID: vaultID, Title: vaultTitle}
	vaultListRes := struct {
		Vaults []client.Vault `json:"vaults"`
		Count  int            `json:"count"`
	}{Vaults: []client.Vault{v}, Count: 1}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v0/users/"+userID+"/vaults":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(&v)
		case r.Method == http.MethodGet && r.URL.Path == "/v0/users/"+userID+"/vaults":
			_ = json.NewEncoder(w).Encode(&vaultListRes)
		case r.Method == http.MethodGet && r.URL.Path == "/v0/users/"+userID+"/vaults/"+vaultTitle:
			_ = json.NewEncoder(w).Encode(&v)
		case r.Method == http.MethodGet && r.URL.Path == "/v0/users/"+userID+"/vaults/"+vaultID:
			_ = json.NewEncoder(w).Encode(&v)
		case r.Method == http.MethodDelete && r.URL.Path == "/v0/users/"+userID+"/vaults/"+vaultID:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	// CreateVault
	created, err := c.CreateVault(ctx, userID, client.CreateVaultRequest{Title: vaultTitle})
	if err != nil {
		t.Fatalf("CreateVault error: %v", err)
	}
	if created.VaultID != vaultID {
		t.Fatalf("vaultId mismatch want %s got %s", vaultID, created.VaultID)
	}

	// ListVaults
	vl, err := c.ListVaults(ctx, userID)
	if err != nil {
		t.Fatalf("ListVaults error: %v", err)
	}
	if len(vl) != 1 || vl[0].VaultID != vaultID {
		t.Fatalf("unexpected vault list %#v", vl)
	}

	// GetVaultByTitle
	byTitle, err := c.GetVaultByTitle(ctx, userID, vaultTitle)
	if err != nil {
		t.Fatalf("GetVaultByTitle error: %v", err)
	}
	if byTitle.VaultID != vaultID {
		t.Fatalf("vaultId mismatch by title")
	}

	// GetVault
	gv, err := c.GetVault(ctx, userID, vaultID)
	if err != nil {
		t.Fatalf("GetVault error: %v", err)
	}
	if gv.VaultID != vaultID {
		t.Fatalf("vault id mismatch")
	}

	// DeleteVault
	if err := c.DeleteVault(ctx, userID, vaultID); err != nil {
		t.Fatalf("DeleteVault error: %v", err)
	}
}
