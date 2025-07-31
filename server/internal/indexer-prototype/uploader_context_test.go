package indexer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// stubEmbedder returns a fixed vector
type stubEmbedder struct{}

func (stubEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func TestUploader_UpsertContexts(t *testing.T) {
	// Setup fake Waviate server
	var batchCalled bool
	var batchBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/batch/objects"):
			body, _ := io.ReadAll(r.Body)
			batchCalled = true
			batchBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			// return empty schema or generic success
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"classes":[]}`))
			} else {
				_, _ = w.Write([]byte(`{}`))
			}
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")

	log := zerolog.Nop()
	up, err := NewUploader(host, log)
	if err != nil {
		t.Fatalf("new uploader: %v", err)
	}

	snaps := []ContextSnapshot{
		{
			UserID:       "u1",
			MemoryID:     "m1",
			ContextID:    "c1",
			CreationTime: time.Now(),
			Text:         "{\"foo\":1}",
		},
	}

	if err := up.UpsertContexts(context.Background(), snaps, stubEmbedder{}); err != nil {
		t.Fatalf("upsert contexts: %v", err)
	}

	if !batchCalled {
		t.Fatalf("batch endpoint not called")
	}
	if !strings.Contains(batchBody, "MemoryContext") {
		t.Fatalf("batch body does not contain MemoryContext class: %s", batchBody)
	}
}
