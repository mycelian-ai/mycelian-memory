package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteMemory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/u1/vaults/v1/memories/m1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL)
	if err := c.DeleteMemory(context.Background(), "u1", "v1", "m1"); err != nil {
		t.Fatalf("delete memory: %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL)
	if err := c.DeleteUser(context.Background(), "u1"); err != nil {
		t.Fatalf("delete user: %v", err)
	}
}
