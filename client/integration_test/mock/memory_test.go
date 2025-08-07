package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_MemoryCRUD(t *testing.T) {
	t.Parallel()

	userID := "u123"
	vaultID := "v1"
	memoryID := "m1"

	m := client.Memory{ID: memoryID, VaultID: vaultID, UserID: userID, Title: "planning", MemoryType: "conversation"}
	memListRes := struct {
		Memories []client.Memory `json:"memories"`
		Count    int             `json:"count"`
	}{Memories: []client.Memory{m}, Count: 1}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(&m)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories":
			_ = json.NewEncoder(w).Encode(&memListRes)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories/"+memoryID:
			_ = json.NewEncoder(w).Encode(&m)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/users/"+userID+"/vaults/"+vaultID+"/memories/"+memoryID:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, client.WithoutExecutor())
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	// CreateMemory
	mem, err := c.CreateMemory(ctx, userID, vaultID, client.CreateMemoryRequest{Title: "planning", MemoryType: "conversation"})
	if err != nil {
		t.Fatalf("CreateMemory error: %v", err)
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

	// GetMemory
	gm, err := c.GetMemory(ctx, userID, vaultID, memoryID)
	if err != nil {
		t.Fatalf("GetMemory error: %v", err)
	}
	if gm.ID != memoryID {
		t.Fatalf("memory id mismatch")
	}

	// DeleteMemory
	if err := c.DeleteMemory(ctx, userID, vaultID, memoryID); err != nil {
		t.Fatalf("DeleteMemory error: %v", err)
	}
}
