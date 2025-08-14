package searchindex

import (
	"context"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/model"
)

// Embeddings produces vector representations for text.
type Embeddings interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Index provides vector search and index maintenance.
type Index interface {
	Search(ctx context.Context, userID, memoryID, query string, vec []float32, topK int, alpha float32) ([]model.SearchHit, error)
	LatestContext(ctx context.Context, userID, memoryID string) (text string, ts time.Time, err error)
	BestContext(ctx context.Context, userID, memoryID, query string, vec []float32, alpha float32) (best string, ts time.Time, score float64, err error)

	// Upserts (best-effort; implementations may ignore or approximate)
	UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error
	UpsertContext(ctx context.Context, contextID string, vec []float32, payload map[string]interface{}) error

	// Synchronous hard-deletes.
	DeleteEntry(ctx context.Context, userID, entryID string) error
	DeleteContext(ctx context.Context, userID, contextID string) error
	DeleteMemory(ctx context.Context, userID, memoryID string) error
	DeleteVault(ctx context.Context, userID, vaultID string) error
}

// HealthPinger is optionally implemented by an Index to expose specialized
// health check logic. Returns nil when healthy.
type HealthPinger interface {
	HealthPing(ctx context.Context) error
}
