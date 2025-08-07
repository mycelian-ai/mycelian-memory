package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

func TestCreateMemory_Success(t *testing.T) {
	t.Parallel()
	want := types.Memory{ID: "m1", VaultID: "v1", UserID: "user_1", Title: "t"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	got, err := CreateMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", types.CreateMemoryRequest{Title: "t", MemoryType: "NOTES"})
	if err != nil || got == nil || got.ID != "m1" {
		t.Fatalf("CreateMemory unexpected: got=%+v err=%v", got, err)
	}
}

func TestListMemories_Success(t *testing.T) {
	t.Parallel()
	resp := types.ListMemoriesResponse{Memories: []types.Memory{{ID: "m1"}}, Count: 1}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	got, err := ListMemories(context.Background(), srv.Client(), srv.URL, "user_1", "v1")
	if err != nil || len(got) != 1 || got[0].ID != "m1" {
		t.Fatalf("ListMemories unexpected: got=%+v err=%v", got, err)
	}
}

func TestGetMemory_Success(t *testing.T) {
	t.Parallel()
	want := types.Memory{ID: "m1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	got, err := GetMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1")
	if err != nil || got == nil || got.ID != "m1" {
		t.Fatalf("GetMemory unexpected: got=%+v err=%v", got, err)
	}
}

func TestDeleteMemory_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := DeleteMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1"); err != nil {
		t.Fatalf("DeleteMemory error: %v", err)
	}
}

func TestMemories_InvalidUserID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := CreateMemory(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1", types.CreateMemoryRequest{Title: "t", MemoryType: "NOTES"}); err == nil {
		t.Fatal("expected validation error for CreateMemory")
	}
	if _, err := ListMemories(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1"); err == nil {
		t.Fatal("expected validation error for ListMemories")
	}
	if _, err := GetMemory(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1", "m1"); err == nil {
		t.Fatal("expected validation error for GetMemory")
	}
	if err := DeleteMemory(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1", "m1"); err == nil {
		t.Fatal("expected validation error for DeleteMemory")
	}
}

func TestMemories_NonOKStatuses(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
		case http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
		case http.MethodDelete:
			w.WriteHeader(http.StatusConflict)
		}
	}))
	defer srv.Close()
	if _, err := CreateMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", types.CreateMemoryRequest{Title: "t", MemoryType: "NOTES"}); err == nil {
		t.Fatal("expected error for CreateMemory non-201")
	}
	if _, err := ListMemories(context.Background(), srv.Client(), srv.URL, "user_1", "v1"); err == nil {
		t.Fatal("expected error for ListMemories non-200")
	}
	if _, err := GetMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1"); err == nil {
		t.Fatal("expected error for GetMemory non-200")
	}
	if err := DeleteMemory(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1"); err == nil {
		t.Fatal("expected error for DeleteMemory non-204")
	}
}

func TestMemories_DecodeErrors(t *testing.T) {
	t.Parallel()
	// CreateMemory decode error
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv1.Close()
	if _, err := CreateMemory(context.Background(), srv1.Client(), srv1.URL, "user_1", "v1", types.CreateMemoryRequest{Title: "t", MemoryType: "NOTES"}); err == nil {
		t.Fatal("expected decode error for CreateMemory")
	}

	// ListMemories decode error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv2.Close()
	if _, err := ListMemories(context.Background(), srv2.Client(), srv2.URL, "user_1", "v1"); err == nil {
		t.Fatal("expected decode error for ListMemories")
	}

	// GetMemory decode error
	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv3.Close()
	if _, err := GetMemory(context.Background(), srv3.Client(), srv3.URL, "user_1", "v1", "m1"); err == nil {
		t.Fatal("expected decode error for GetMemory")
	}
}

func TestMemories_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := CreateMemory(context.Background(), hc, "http://example.com", "user_1", "v1", types.CreateMemoryRequest{Title: "t", MemoryType: "NOTES"}); err == nil {
		t.Fatal("expected Do error for CreateMemory")
	}
	if _, err := ListMemories(context.Background(), hc, "http://example.com", "user_1", "v1"); err == nil {
		t.Fatal("expected Do error for ListMemories")
	}
	if _, err := GetMemory(context.Background(), hc, "http://example.com", "user_1", "v1", "m1"); err == nil {
		t.Fatal("expected Do error for GetMemory")
	}
	if err := DeleteMemory(context.Background(), hc, "http://example.com", "user_1", "v1", "m1"); err == nil {
		t.Fatal("expected Do error for DeleteMemory")
	}
}
