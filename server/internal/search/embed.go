package search

import "context"

// Embedder abstracts embedding generation.
// Returned slice must be non-nil and contain at least 1 dimension.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// compile-time proof that OllamaProvider implements Embedder
var _ Embedder = (*OllamaProvider)(nil)
