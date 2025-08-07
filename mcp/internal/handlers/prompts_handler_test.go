package handlers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mycelian/mycelian-memory/client"
	"github.com/mycelian/mycelian-memory/client/prompts"
)

func TestGetDefaultPromptsTool_MatchesEmbedded(t *testing.T) {
	memTypes, err := prompts.ListMemoryTypes()
	if err != nil {
		t.Fatalf("list memory types: %v", err)
	}
	if len(memTypes) == 0 {
		t.Fatalf("no memory types found in embedded prompts")
	}

	sdk := client.New("http://example.com")
	ph := NewPromptsHandler(sdk)

	for _, mt := range memTypes {
		t.Run(mt, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
				"memory_type": mt,
			}}}

			res, err := ph.handleGetPrompts(context.Background(), req)
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if res == nil || res.IsError {
				t.Fatalf("tool result error: %+v", res)
			}

			// Extract JSON string
			txt := res.Content[0].(mcp.TextContent).Text
			var got prompts.DefaultPromptResponse
			if err := json.Unmarshal([]byte(txt), &got); err != nil {
				t.Fatalf("unmarshal result: %v", err)
			}

			want, _ := prompts.LoadDefaultPrompts(mt)
			if !reflect.DeepEqual(&got, want) {
				t.Fatalf("payload mismatch for %s", mt)
			}
		})
	}
}
