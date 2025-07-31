package indexer

import (
	"context"
	"fmt"
)

// Embedder abstracts embedding generation.
// Returned slice must be non-nil and contain at least 1 dimension.

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewProvider returns an Embedder implementation based on name.
func NewProvider(name, model string) (Embedder, error) {
	switch name {
	case "ollama":
		return NewOllamaProvider(model), nil
	case "openai":
		return nil, fmt.Errorf("openai provider not implemented yet")
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
