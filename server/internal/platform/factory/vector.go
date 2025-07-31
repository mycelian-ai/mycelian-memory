package factory

import (
	"memory-backend/internal/config"
	"memory-backend/internal/search"
)

// NewVectorStore selects the appropriate search adapter based on cfg.VectorStore.
// Waviate is the sole vector store.
func NewVectorStore(cfg *config.Config) (search.Searcher, error) {
	return search.NewWaviateSearcher(cfg.WaviateURL)
}
