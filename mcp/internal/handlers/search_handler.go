package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
)

// SearchHandler exposes the search_memories tool.
type SearchHandler struct {
	client *client.Client
}

func NewSearchHandler(c *client.Client) *SearchHandler {
	return &SearchHandler{client: c}
}

// RegisterTools registers the search_memories tool.
func (sh *SearchHandler) RegisterTools(s *server.MCPServer) error {
	searchTool := mcp.NewTool("search_memories",
		mcp.WithDescription("Hybrid semantic + keyword search within a memory. Results include:\n • entries – entry hits (controlled by ke).\n • latestContext – the most recent consolidated context snapshot (string).\n • bestContext – the context snapshot that most closely matches the query, if found, plus score & timestamp.\n\nParameters:\n • memory_id (required) – target memory.\n • query (required) – search text.\n • top_k (optional) – legacy combined top-k (1–100); retained for back-compat.\n • ke (optional) – top-k for entries (recommended 5).\n • kc (optional) – top-k for context shards (recommended 3)."),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
		mcp.WithNumber("top_k", mcp.Description("Legacy combined top-k (1–100); prefer ke/kc")),
		mcp.WithNumber("ke", mcp.Description("Top-k for entries (recommended 5)")),
		mcp.WithNumber("kc", mcp.Description("Top-k for context shards (recommended 3)")),
	)
	s.AddTool(searchTool, sh.handleSearch)
	return nil
}

func (sh *SearchHandler) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memoryID, _ := req.RequireString("memory_id")
	query, _ := req.RequireString("query")

	topK := 10
	if v, ok := req.GetArguments()["top_k"].(float64); ok {
		if v >= 1 && v <= 100 {
			topK = int(v)
		}
	}

	resp, err := sh.client.Search(ctx, client.SearchRequest{
		MemoryID: memoryID,
		Query:    query,
		TopK:     topK,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Build payload preserving raw JSON fields; use camelCase to match client/docs.
	payload := map[string]interface{}{
		"entries":              resp.Entries,
		"count":                resp.Count,
		"latestContext":        json.RawMessage(resp.LatestContext),
		"contextTimestamp":     resp.ContextTimestamp,
		"bestContext":          json.RawMessage(resp.BestContext),
		"bestContextTimestamp": resp.BestContextTimestamp,
		"bestContextScore":     resp.BestContextScore,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
