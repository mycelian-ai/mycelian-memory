package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type mockEmbedder struct{}

func (mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.0}, nil
}

func TestIndexerOnce(t *testing.T) {
	cfg := Config{Interval: 1 * time.Second, Once: true}
	logger := zerolog.Nop()
	idx := New(cfg, mockEmbedder{}, nil, nil, &State{}, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := idx.Run(ctx); err != nil {
		t.Fatalf("indexer run returned error: %v", err)
	}
}
