package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	weaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	filters "github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	gql "github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"

	"memory-backend/internal/indexer-prototype"
)

func main() {
	var (
		query      string
		waviateURL string
		topK       int
		provider   string
		model      string
		userID     string
		memoryID   string
	)

	flag.StringVar(&query, "q", "", "Query text (required)")
	flag.StringVar(&query, "query", "", "Query text (required, long form)")
	flag.StringVar(&waviateURL, "waviate-url", "http://localhost:8082", "Waviate base URL")
	flag.IntVar(&topK, "k", 5, "Top K results")
	flag.IntVar(&topK, "topk", 5, "Top K results (long form)")
	flag.StringVar(&provider, "provider", "ollama", "Embedding provider (ollama|openai)")
	flag.StringVar(&provider, "p", "ollama", "Embedding provider shorthand")
	flag.StringVar(&model, "embed-model", "mxbai-embed-large", "Embedding model name")
	flag.StringVar(&model, "e", "mxbai-embed-large", "Embedding model shorthand")
	flag.StringVar(&userID, "user", "", "Tenant userId (optional)")
	flag.StringVar(&memoryID, "memory", "", "Filter by memoryId (optional, requires --user)")
	flag.Parse()

	if query == "" {
		fmt.Println("-q query is required")
		os.Exit(1)
	}
	if memoryID != "" && userID == "" {
		fmt.Println("--memory filter requires --user tenant specified")
		os.Exit(1)
	}

	// Embed query
	emb, err := indexer.NewProvider(provider, model)
	if err != nil {
		log.Fatal().Err(err).Msg("embedder init failed")
	}
	vec, err := emb.Embed(context.Background(), query)
	if err != nil {
		log.Fatal().Err(err).Msg("embedding failed")
	}

	// Waviate client
	cfg := weaviate.Config{Scheme: "http", Host: waviateURL}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("waviate client")
	}

	hy := (&gql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithVector(vec).
		WithAlpha(0.6)

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
		log.Fatal().Err(err).Msg("query failed")
	}

	out, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(out))
}
