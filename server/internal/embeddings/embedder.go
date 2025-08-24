package embeddings

import "context"

// Provider produces vector representations for text.
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}
