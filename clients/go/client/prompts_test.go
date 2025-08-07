package client

import (
    "context"
    "testing"
)

func TestLoadDefaultPrompts(t *testing.T) {
    c := New("http://example.com")
    p, err := c.LoadDefaultPrompts(context.Background(), "chat")
    if err != nil {
        t.Fatalf("LoadDefaultPrompts: %v", err)
    }
    if p == nil {
        t.Fatalf("expected prompts")
    }
}
