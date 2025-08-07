package api

import (
	"context"
	"testing"
)

func TestLoadDefaultPrompts_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp, err := LoadDefaultPrompts(ctx, "chat")
	if err != nil || resp == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Version == "" || len(resp.Templates) == 0 || resp.ContextSummaryRules == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestLoadDefaultPrompts_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadDefaultPrompts(ctx, "chat"); err == nil {
		t.Fatal("expected context error")
	}
}
