package wavtool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	filters "github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"

	"github.com/mycelian/mycelian-memory/server/internal/search"
)

// Query performs a hybrid search against Waviate and returns the raw JSON response bytes.
// If memoryID is provided, results are filtered to that memory. If userID is provided,
// the tenant is scoped to that user.
func Query(baseURL, provider, model, userID, memoryID, query string, topK int, alpha float32) ([]byte, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if topK <= 0 {
		topK = 5
	}

	// Embed query
	emb, err := search.NewProvider(provider, model)
	if err != nil {
		return nil, fmt.Errorf("embedder init failed: %w", err)
	}
	vec, err := emb.Embed(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	// Waviate client
	cfg := weaviate.Config{Scheme: "http", Host: baseURL}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("waviate client: %w", err)
	}

	hy := (&gql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithVector(vec).
		WithAlpha(alpha)

	builder := client.GraphQL().Get().WithClassName("MemoryEntry").WithHybrid(hy).WithLimit(topK).
		WithFields(
			gql.Field{Name: "entryId"},
			gql.Field{Name: "userId"},
			gql.Field{Name: "memoryId"},
			gql.Field{Name: "summary"},
			gql.Field{Name: "rawEntry"},
			gql.Field{Name: "_additional", Fields: []gql.Field{{Name: "score"}}},
		)

	if userID != "" {
		builder = builder.WithTenant(userID)
	}
	if memoryID != "" {
		where := filters.Where().WithPath([]string{"memoryId"}).WithOperator(filters.Equal).WithValueText(memoryID)
		builder = builder.WithWhere(where)
	}

	resp, err := builder.Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Warn().Err(err).Msg("marshal response indent failed; falling back to compact")
		return json.Marshal(resp)
	}
	return out, nil
}
