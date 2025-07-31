package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestVaultEndpoints(t *testing.T) {
	userID := "user-123"
	vaultID := "vault-456"
	vaultTitle := "work-projects"

	// Prepare canned responses
	v := Vault{UserID: userID, VaultID: vaultID, Title: vaultTitle}
	vaultListRes := struct {
		Vaults []Vault `json:"vaults"`
		Count  int     `json:"count"`
	}{Vaults: []Vault{v}, Count: 1}

	memoryID := "mem-789"
	m := Memory{ID: memoryID, VaultID: vaultID, UserID: userID, Title: "planning", MemoryType: "conversation"}
	memListRes := struct {
		Memories []Memory `json:"memories"`
		Count    int      `json:"count"`
	}{Memories: []Memory{m}, Count: 1}

	// mux handler to differentiate requests
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/users/"+userID+"/vaults":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(&v)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults":
			_ = json.NewEncoder(w).Encode(&vaultListRes)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultTitle:
			_ = json.NewEncoder(w).Encode(&v)
		case r.Method == http.MethodPost && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(&m)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories":
			_ = json.NewEncoder(w).Encode(&memListRes)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultTitle+"/memories/planning":
			_ = json.NewEncoder(w).Encode(&m)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	defer srv.Close()

	c := New(srv.URL, WithoutExecutor()) // sync client is enough for these tests
	ctx := context.Background()

	// CreateVault
	created, err := c.CreateVault(ctx, userID, CreateVaultRequest{Title: vaultTitle})
	if err != nil {
		t.Fatalf("CreateVault error: %v", err)
	}
	if !reflect.DeepEqual(created.VaultID, vaultID) {
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

	// CreateMemoryInVault
	mem, err := c.CreateMemoryInVault(ctx, userID, vaultID, CreateMemoryRequest{Title: "planning", MemoryType: "conversation"})
	if err != nil {
		t.Fatalf("CreateMemoryInVault error: %v", err)
	}
	if mem.ID != memoryID {
		t.Fatalf("memory id mismatch")
	}

	// ListMemories
	ml, err := c.ListMemories(ctx, userID, vaultID)
	if err != nil {
		t.Fatalf("ListMemories error: %v", err)
	}
	if len(ml) != 1 || ml[0].ID != memoryID {
		t.Fatalf("unexpected memory list %#v", ml)
	}

	// GetMemoryByTitle
	mByTitle, err := c.GetMemoryByTitle(ctx, userID, vaultTitle, "planning")
	if err != nil {
		t.Fatalf("GetMemoryByTitle error: %v", err)
	}
	if mByTitle.ID != memoryID {
		t.Fatalf("memory id mismatch by title")
	}
}
