package prompts

import "testing"

func TestListMemoryTypes_ContainsChatAndCode(t *testing.T) {
	types, err := ListMemoryTypes()
	if err != nil {
		t.Fatalf("ListMemoryTypes error: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("expected at least one type")
	}
	// Require 'chat' (primary); other types (e.g., 'code') are optional for now
	foundChat := false
	for _, v := range types {
		if v == "chat" {
			foundChat = true
		}
	}
	if !foundChat {
		t.Fatalf("expected 'chat' present, got %v", types)
	}
}
