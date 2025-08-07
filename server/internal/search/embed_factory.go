package search

import (
	"fmt"

	idx "github.com/mycelian/mycelian-memory/server/internal/indexer-prototype"
)

// NewProvider returns an Embedder for the given provider/model using the
// same implementation as the indexer. It keeps the search package decoupled
// from concrete embedding providers.
func NewProvider(provider, model string) (Embedder, error) {
	e, err := idx.NewProvider(provider, model)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, fmt.Errorf("provider %s returned nil", provider)
	}
	return e, nil
}
