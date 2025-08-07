package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/clients/go/client"
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
		mcp.WithDescription("Hybrid semantic + keyword search within a memory. Results include:\n • entries – top-K entry hits.\n • latestContext – the most recent consolidated context snapshot (string).\n • bestContext – the context snapshot that most closely matches the query, if found, plus score & timestamp."),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("The UUID of the user")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
		mcp.WithNumber("top_k", mcp.Description("Number of results to return (1-100, default 10)")),
	)
	s.AddTool(searchTool, sh.handleSearch)
	return nil
}

func (sh *SearchHandler) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userID, _ := req.RequireString("user_id")
	memoryID, _ := req.RequireString("memory_id")
	query, _ := req.RequireString("query")

	topK := 10
	if v, ok := req.GetArguments()["top_k"].(float64); ok {
		if v >= 1 && v <= 100 {
			topK = int(v)
		}
	}

	resp, err := sh.client.Search(ctx, client.SearchRequest{
		UserID:   userID,
		MemoryID: memoryID,
		Query:    query,
		TopK:     topK,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Build payload preserving LatestContext raw JSON.
	payload := map[string]interface{}{
		"entries":           resp.Entries,
		"count":             resp.Count,
		"latest_context":    json.RawMessage(resp.LatestContext),
		"context_timestamp": resp.ContextTimestamp,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
