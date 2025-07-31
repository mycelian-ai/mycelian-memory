package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_CreateUser(t *testing.T) {
	// Prepare fake backend
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

	c := New(hs.URL)
	ctx := context.Background()
	user, err := c.CreateUser(ctx, CreateUserRequest{UserID: "u123", Email: "test@example.com", DisplayName: "Test User"})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.ID != "u123" || user.Email != "test@example.com" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestClient_GetUser(t *testing.T) {
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

	c := New(hs.URL)
	ctx := context.Background()
	user, err := c.GetUser(ctx, "u123")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user.ID != "u123" || user.Email != "test@example.com" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestClient_CreateMemory(t *testing.T) {
	// parse expected time for comparison
	expTime, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")

	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/users/u123/memories" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"memoryId":"m789",
			"userId":"u123",
			"title":"Project Alpha",
			"description":"Alpha description",
			"memory_type":"PROJECT",
			"created_at":"2025-01-01T00:00:00Z",
			"updated_at":"2025-01-01T00:00:00Z"
		}`))
	}))
	defer hs.Close()

	c := New(hs.URL)
	ctx := context.Background()
	memReq := CreateMemoryRequest{Title: "Project Alpha", Description: "Alpha description", MemoryType: "PROJECT"}
	memory, err := c.CreateMemory(ctx, "u123", memReq)
	if err != nil {
		t.Fatalf("CreateMemory returned error: %v", err)
	}
	if memory.ID != "m789" || memory.UserID != "u123" || !memory.CreatedAt.Equal(expTime) {
		t.Fatalf("unexpected memory %+v", memory)
	}
}
