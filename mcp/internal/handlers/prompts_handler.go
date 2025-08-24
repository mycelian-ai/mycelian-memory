package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
)

// PromptsHandler exposes the get_default_prompts tool.
// It returns embedded default prompt templates for a given memory type.
type PromptsHandler struct {
	client *client.Client
}

func NewPromptsHandler(c *client.Client) *PromptsHandler {
	return &PromptsHandler{client: c}
}

// RegisterTools registers the get_default_prompts tool on the MCP server.
func (ph *PromptsHandler) RegisterTools(s *server.MCPServer) error {
	tool := mcp.NewTool("get_default_prompts",
		mcp.WithDescription("Return default prompt templates for a given memory type"),
		mcp.WithString("memory_type", mcp.Required(), mcp.Description("Memory type, e.g. chat, code")),
	)
	s.AddTool(tool, ph.handleGetPrompts)
	return nil
}

func (ph *PromptsHandler) handleGetPrompts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	memType, _ := req.RequireString("memory_type")

	resp, err := ph.client.LoadDefaultPrompts(ctx, memType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get_default_prompts failed: %v", err)), nil
	}

	b, _ := json.MarshalIndent(resp, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
