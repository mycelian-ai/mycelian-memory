package client

import (
	"context"
	"reflect"
	"testing"

	"github.com/synapse/synapse-mcp-server/prompts"
)

func TestClient_GetDefaultPrompts(t *testing.T) {
	c := MustNew("http://example.com")

	got, err := c.GetDefaultPrompts(context.Background(), "code")
	if err != nil {
		t.Fatalf("GetDefaultPrompts error: %v", err)
	}

	want, _ := prompts.LoadDefaultPrompts("code")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch between SDK and loader:\nwant %+v\n got %+v", want, got)
	}
}

func TestClient_GetDefaultPrompts_ContextCanceled(t *testing.T) {
	c := MustNew("http://example.com")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetDefaultPrompts(ctx, "chat")
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
