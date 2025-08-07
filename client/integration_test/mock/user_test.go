package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_CreateUser_Success(t *testing.T) {
	t.Parallel()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/users" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
            "userId":"u123",
            "email":"test@example.com",
            "displayName":"Test User",
            "timeZone":"UTC",
            "created_at":"2025-01-01T00:00:00Z",
            "updated_at":"2025-01-01T00:00:00Z"
        }`))
	}))
	defer hs.Close()

	c := client.New(hs.URL)
	t.Cleanup(func() { _ = c.Close() })
	user, err := c.CreateUser(context.Background(), client.CreateUserRequest{UserID: "u123", Email: "test@example.com", DisplayName: "Test User"})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.ID != "u123" || user.Email != "test@example.com" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestClient_GetUser_Success(t *testing.T) {
	t.Parallel()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/users/u123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
            "userId":"u123",
            "email":"test@example.com",
            "displayName":"Test User",
            "timeZone":"UTC",
            "created_at":"2025-01-01T00:00:00Z",
            "updated_at":"2025-01-01T00:00:00Z"
        }`))
	}))
	defer hs.Close()

	c := client.New(hs.URL)
	t.Cleanup(func() { _ = c.Close() })
	user, err := c.GetUser(context.Background(), "u123")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user.ID != "u123" || user.Email != "test@example.com" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestClient_DeleteUser_Success(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := client.New(srv.URL)
	t.Cleanup(func() { _ = c.Close() })
	if err := c.DeleteUser(context.Background(), "u1"); err != nil {
		t.Fatalf("delete user: %v", err)
	}
}
