package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// assetMap defines logical IDs → embedded file paths.
var assetMap = map[string]string{
	"ctx_rules":           "prompts/system/context_summary_rules.md",
	"ctx_prompt_chat":     "prompts/default/chat/context_prompt.md",
	"entry_prompt_chat":   "prompts/default/chat/entry_capture_prompt.md",
	"summary_prompt_chat": "prompts/default/chat/summary_prompt.md",
}

// AssetHandler exposes list_assets and get_asset tools.
type AssetHandler struct{}

func NewAssetHandler() *AssetHandler { return &AssetHandler{} }

func (h *AssetHandler) RegisterTools(s *server.MCPServer) error {
	// list_assets has no parameters
	listTool := mcp.NewTool(
		"list_assets",
		mcp.WithDescription("Return logical IDs of static assets available via get_asset"),
	)
	s.AddTool(listTool, h.handleListAssets)

	getTool := mcp.NewTool(
		"get_asset",
		mcp.WithDescription("Return raw text content of a prompt / rule asset"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Logical asset ID")),
	)
	s.AddTool(getTool, h.handleGetAsset)
	return nil
}

func (h *AssetHandler) handleListAssets(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ids := make([]string, 0, len(assetMap))
	for id := range assetMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	b, _ := json.MarshalIndent(struct {
		Assets []string `json:"assets"`
	}{Assets: ids}, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (h *AssetHandler) handleGetAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, _ := req.RequireString("id")
	path, ok := assetMap[id]
	if !ok {
		return mcp.NewToolResultError("unknown asset id"), nil
	}
	// Resolve path relative to executable working directory.
	abs, _ := filepath.Abs(path)
	data, err := os.ReadFile(abs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read error: %v", err)), nil
	}
	// Return as plain text (not JSON) so model gets raw markdown.
	return mcp.NewToolResultText(string(data)), nil
}
