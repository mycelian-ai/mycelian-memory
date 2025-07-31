package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type embedResp struct {
	Embedding []float64 `json:"embedding"`
}

func TestOllamaProvider(t *testing.T) {
	// create fake server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(embedResp{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer srv.Close()

	p := NewOllamaProvider("dummy-model")
	// override base URL for test
	p.client.SetBaseURL(srv.URL)

	vec, err := p.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3 dims, got %d", len(vec))
	}
}
