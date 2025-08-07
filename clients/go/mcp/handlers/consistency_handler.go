package handlers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/clients/go/client"
)

// ConsistencyHandler exposes await_consistency tool.
type ConsistencyHandler struct {
	client *client.Client
}

func NewConsistencyHandler(c *client.Client) *ConsistencyHandler {
	return &ConsistencyHandler{client: c}
}

func (h *ConsistencyHandler) RegisterTools(s *server.MCPServer) error {
	awaitTool := mcp.NewTool("await_consistency",
		mcp.WithDescription(`Block until all queued writes for the given memory have finished executing on the MCP shard-queue.

Typical use-cases:
• After a sequence of put_context / add_entry calls when the agent needs a strong read-after-write guarantee.
• Before issuing get_context or list_entries that must reflect the very latest state.

Example:
  1. call put_context(user_id="u", memory_id="m", content="draft v2")
  2. call add_entry(...)
  3. call await_consistency(user_id="u", memory_id="m")   # returns "consistent"
  4. call get_context(...) – guaranteed to see "draft v2"`),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID (unused)")),
		mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory UUID")),
	)
	s.AddTool(awaitTool, h.handleAwait)
	return nil
}

func (h *ConsistencyHandler) handleAwait(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memID, _ := req.RequireString("memory_id")
	// user_id not required by SDK helper
	if err := h.client.AwaitConsistency(ctx, memID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("await consistency failed: %v", err)), nil
	}
	return mcp.NewToolResultText("consistent"), nil
}
