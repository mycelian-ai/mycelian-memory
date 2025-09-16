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
		mcp.WithDescription(`Performs hybrid semantic and keyword search within a memory.

Parameters:
• memory_id (required): Target memory UUID
• query (required): Search query text
• top_ke (optional): Number of entry results to return
  - Default: 5
  - Range: 0-25 (0 returns no entries, useful for context-only searches)
• top_kc (optional): Number of context shard results to return
  - Default: 2
  - Range: 1-10 (must be at least 1)
• include_raw_entries (optional): Include raw entry content in results
  - Default: false (raw entries excluded to save tokens)
  - Set to true to include full raw entry content

Returns:
• entries: Array of matching entries (size: 0 to top_ke), each with:
  - entryId, summary, rawEntry, score, creationTime
• count: Number of entries returned
• latestContext: The most recent context snapshot (always present)
• latestContextTimestamp: Timestamp of the latest context (always present)
• contexts: Array of semantically matching context shards (size: 0 to top_kc), each with:
  - context: Text content
  - timestamp: When this context was created
  - score: Relevance score (0-1)

The timestamps allow understanding temporal evolution of the memory. Context shards are sorted by relevance score descending. Entries are sorted by relevance score descending.`),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("The UUID of the memory")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query text")),
		mcp.WithNumber("top_ke", mcp.Description("Top-k for entries (default: 5, range: 0-25)")),
		mcp.WithNumber("top_kc", mcp.Description("Top-k for context shards (default: 2, range: 1-10)")),
		mcp.WithBoolean("include_raw_entries", mcp.Description("Include raw entry content in results (default: false)")),
	)
	s.AddTool(searchTool, sh.handleSearch)
	return nil
}

func (sh *SearchHandler) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memoryID, _ := req.RequireString("memory_id")
	query, _ := req.RequireString("query")

	// Handle top_ke parameter (default: 5, range: 0-25)
	topKE := 5
	if v, ok := req.GetArguments()["top_ke"].(float64); ok {
		topKE = int(v)
		if topKE < 0 || topKE > 25 {
			return mcp.NewToolResultError("top_ke must be between 0 and 25"), nil
		}
	}

	// Handle top_kc parameter (default: 2, range: 1-10)
	topKC := 2
	if v, ok := req.GetArguments()["top_kc"].(float64); ok {
		topKC = int(v)
		if topKC < 1 || topKC > 10 {
			return mcp.NewToolResultError("top_kc must be between 1 and 10"), nil
		}
	}

	// Handle include_raw_entries parameter (default: false)
	includeRawEntries := false
	if v, ok := req.GetArguments()["include_raw_entries"].(bool); ok {
		includeRawEntries = v
	}

	resp, err := sh.client.Search(ctx, client.SearchRequest{
		MemoryID:          memoryID,
		Query:             query,
		TopKE:             &topKE,
		TopKC:             &topKC,
		IncludeRawEntries: includeRawEntries,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Build payload preserving raw JSON fields; use camelCase to match client/docs.
	payload := map[string]interface{}{
		"entries":                resp.Entries,
		"count":                  resp.Count,
		"latestContext":          json.RawMessage(resp.LatestContext),
		"latestContextTimestamp": resp.LatestContextTimestamp,
		"contexts":               resp.Contexts,
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
