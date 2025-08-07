package factory

import (
	"github.com/mycelian/mycelian-memory/server/internal/config"
	"github.com/mycelian/mycelian-memory/server/internal/search"
)

// NewVectorStore selects the appropriate search adapter based on cfg.VectorStore.
// Waviate is the sole vector store.
func NewVectorStore(cfg *config.Config) (search.Searcher, error) {
	return search.NewWaviateSearcher(cfg.WaviateURL)
}
