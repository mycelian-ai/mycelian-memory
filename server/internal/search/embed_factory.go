package search

import (
	"fmt"
)

// NewProvider returns an Embedder for the given provider/model using the
// internal providers. It keeps the search package decoupled from
// concrete embedding providers.
func NewProvider(provider, model string) (Embedder, error) {
	switch provider {
	case "ollama":
		e := NewOllamaProvider(model)
		if e == nil {
			return nil, fmt.Errorf("provider %s returned nil", provider)
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}
