package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

func TestSearch_Success(t *testing.T) {
	t.Parallel()
	want := types.SearchResponse{Count: 1}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	got, err := Search(context.Background(), srv.Client(), srv.URL, types.SearchRequest{MemoryID: "m1", Query: "q"})
	if err != nil || got == nil || got.Count != 1 {
		t.Fatalf("Search unexpected: %+v, err=%v", got, err)
	}
}

func TestSearch_NonOKAndDecodeError(t *testing.T) {
	t.Parallel()
	// Non-OK
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv1.Close()
	if _, err := Search(context.Background(), srv1.Client(), srv1.URL, types.SearchRequest{MemoryID: "m", Query: "q"}); err == nil {
		t.Fatal("expected error for non-OK status")
	}

	// Decode error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv2.Close()
	if _, err := Search(context.Background(), srv2.Client(), srv2.URL, types.SearchRequest{MemoryID: "m", Query: "q"}); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestSearch_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := Search(context.Background(), hc, "http://example.com", types.SearchRequest{MemoryID: "m", Query: "q"}); err == nil {
		t.Fatal("expected Do error for Search")
	}
}
