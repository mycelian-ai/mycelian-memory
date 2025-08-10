package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mycelian/mycelian-memory/server/wavtool"
)

func main() {
	var (
		query    string
		wavURL   string
		topK     int
		provider string
		model    string
		userID   string
		memoryID string
	)

	flag.StringVar(&query, "q", "", "Query text (required)")
	flag.StringVar(&query, "query", "", "Query text (required, long form)")
	flag.StringVar(&wavURL, "waviate-url", "http://localhost:8082", "Waviate base URL")
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

	out, err := wavtool.Query(wavURL, provider, model, userID, memoryID, query, topK, 0.6)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
