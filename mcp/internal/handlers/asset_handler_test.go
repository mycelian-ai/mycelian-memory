package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestAssetHandler_ListAndGet(t *testing.T) {
	ah := NewAssetHandler()

	// Test list_assets
	listRes, err := ah.handleListAssets(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("list_assets returned error: %v", err)
	}
	var payload struct {
		Assets []string `json:"assets"`
	}
	if err := json.Unmarshal([]byte(listRes.Content[0].(mcp.TextContent).Text), &payload); err != nil {
		t.Fatalf("failed to decode list_assets payload: %v", err)
	}
	if len(payload.Assets) == 0 {
		t.Fatalf("expected non-empty asset list")
	}

	// Sanity check: each asset id can be fetched
	for _, id := range payload.Assets {
		res, err := ah.handleGetAsset(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"id": id}}})
		if err != nil {
			t.Fatalf("get_asset(%s) error: %v", id, err)
		}
		text := res.Content[0].(mcp.TextContent).Text
		if text == "" {
			t.Fatalf("get_asset(%s) returned empty text", id)
		}
	}
}
