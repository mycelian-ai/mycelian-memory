package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCLI_CreateVaultMemoryEntry_ListEntries(t *testing.T) {
	// Test updated to work with dev mode auth (no --user-id flags needed)
	// Stub backend for dev mode auth
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/vaults", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"vaultId": "vault-999",
				"title":   "TestVault",
			})
		}
	})
	mux.HandleFunc("/v0/vaults/vault-999/memories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"memoryId": "mem-456",
				"vaultId":  "vault-999",
			})
		}
	})
	mux.HandleFunc("/v0/vaults/vault-999/memories/mem-456/entries", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"entryId":  "entry-789",
				"userId":   "user-123",
				"memoryId": "mem-456",
			})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"entries": []map[string]string{{
					"entryId":  "entry-789",
					"rawEntry": "hello",
				}},
				"count": 1,
			})
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := os.Setenv("MEMORY_SERVICE_URL", srv.URL); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	root := NewRootCmd()

	// Note: create-user command doesn't exist - users are managed via API
	// Start with create-vault, then create-memory
	root.SetArgs([]string{"create-vault", "--service-url", srv.URL, "--title", "TestVault"})
	if err := root.Execute(); err != nil {
		t.Fatalf("create-vault cmd failed: %v", err)
	}

	// create-memory
	root.SetArgs([]string{"create-memory", "--service-url", srv.URL, "--vault-id", "vault-999", "--title", "Test", "--memory-type", "PROJECT"})
	if err := root.Execute(); err != nil {
		t.Fatalf("create-memory cmd failed: %v", err)
	}

	// create-entry
	root.SetArgs([]string{"create-entry", "--service-url", srv.URL, "--vault-id", "vault-999", "--memory-id", "mem-456", "--raw-entry", "hello", "--summary", "hello summary"})
	if err := root.Execute(); err != nil {
		t.Fatalf("create-entry cmd failed: %v", err)
	}

	// list-entries
	b := &strings.Builder{}
	rootList := NewRootCmd()
	rootList.SetOut(b)
	rootList.SetArgs([]string{"list-entries", "--service-url", srv.URL, "--vault-id", "vault-999", "--memory-id", "mem-456"})
	if err := rootList.Execute(); err != nil {
		t.Fatalf("list-entries cmd failed: %v", err)
	}
	_ = b

	// list-entries limit=1 (formerly top-entries)
	b2 := &strings.Builder{}
	rootTop := NewRootCmd()
	rootTop.SetOut(b2)
	rootTop.SetArgs([]string{"list-entries", "--service-url", srv.URL, "--vault-id", "vault-999", "--memory-id", "mem-456", "--limit", "1"})
	if err := rootTop.Execute(); err != nil {
		t.Fatalf("list-entries cmd failed: %v", err)
	}
}
