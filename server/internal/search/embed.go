package search

import idx "github.com/mycelian/mycelian-memory/server/internal/indexer-prototype"

// Embedder exposes text embedding capability for the search service.
// It is an alias to the Embedder interface used by the indexer so that
// both components rely on the same abstraction without code duplication.
//
//     type Embedder interface {
//         Embed(ctx context.Context, text string) ([]float32, error)
//     }
//
// Any provider that satisfies idx.Embedder automatically satisfies this alias.
// This indirection avoids import cycles between packages.
//
// Compile-time assertion below guarantees the aliasing works.

type Embedder = idx.Embedder

// compile-time proof that OllamaProvider implements Embedder
var _ Embedder = (*idx.OllamaProvider)(nil)
