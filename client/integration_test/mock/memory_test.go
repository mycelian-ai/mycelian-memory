package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
	"github.com/mycelian/mycelian-memory/devmode"
)

func TestClient_MemoryCRUD(t *testing.T) {
	t.Parallel()

	userID := "mycelian-dev" // This matches the actor ID that the server resolves for the local dev API key
	vaultID := "v1"
	memoryID := "m1"

	m := client.Memory{ID: memoryID, VaultID: vaultID, UserID: userID, Title: "planning", MemoryType: "conversation"}
	memListRes := struct {
		Memories []client.Memory `json:"memories"`
		Count    int             `json:"count"`
	}{Memories: []client.Memory{m}, Count: 1}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Check Authorization header
		if r.Header.Get("Authorization") != "Bearer "+devmode.APIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v0/vaults/"+vaultID+"/memories":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(&m)
		case r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/"+vaultID+"/memories":
			_ = json.NewEncoder(w).Encode(&memListRes)
		case r.Method == http.MethodGet && r.URL.Path == "/v0/vaults/"+vaultID+"/memories/"+memoryID:
			_ = json.NewEncoder(w).Encode(&m)
		case r.Method == http.MethodDelete && r.URL.Path == "/v0/vaults/"+vaultID+"/memories/"+memoryID:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	defer srv.Close()

	c := client.NewWithDevMode(srv.URL)
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	// CreateMemory
	mem, err := c.CreateMemory(ctx, vaultID, client.CreateMemoryRequest{Title: "planning", MemoryType: "conversation"})
	if err != nil {
		t.Fatalf("CreateMemory error: %v", err)
	}
	if mem.ID != memoryID {
		t.Fatalf("memory id mismatch")
	}

	// ListMemories
	ml, err := c.ListMemories(ctx, vaultID)
	if err != nil {
		t.Fatalf("ListMemories error: %v", err)
	}
	if len(ml) != 1 || ml[0].ID != memoryID {
		t.Fatalf("unexpected memory list %#v", ml)
	}

	// GetMemory
	gm, err := c.GetMemory(ctx, vaultID, memoryID)
	if err != nil {
		t.Fatalf("GetMemory error: %v", err)
	}
	if gm.ID != memoryID {
		t.Fatalf("memory id mismatch")
	}

	// DeleteMemory
	if err := c.DeleteMemory(ctx, vaultID, memoryID); err != nil {
		t.Fatalf("DeleteMemory error: %v", err)
	}
}
