package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mock weaviate server responding with provided body
func newMockServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func TestSearch_NilMemoryEntryReturnsEmpty(t *testing.T) {
	srv := newMockServer(`{"data":{"Get":{"MemoryEntry":null}}}`)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	s, err := NewWaviateSearcher(host)
	if err != nil {
		t.Fatalf("new searcher: %v", err)
	}

	res, err := s.Search(context.Background(), "u1", "m1", "q", []float32{1}, 5, 0.6)
	if err != nil {
		t.Fatalf("search err: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected 0 results, got %d", len(res))
	}
}

func TestSearch_TenantNotFoundError(t *testing.T) {
	srv := newMockServer(`{"data":{"Get":{"MemoryEntry":null}},"errors":[{"message":"tenant not found: \"u1\""}]}`)
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")
	s, _ := NewWaviateSearcher(host)

	_, err := s.Search(context.Background(), "u1", "m1", "q", []float32{1}, 5, 0.6)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
