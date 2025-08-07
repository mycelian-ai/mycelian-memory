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
	// Expect "chat" and "code" based on embedded dirs
	foundChat, foundCode := false, false
	for _, v := range types {
		if v == "chat" {
			foundChat = true
		}
		if v == "code" {
			foundCode = true
		}
	}
	if !foundChat || !foundCode {
		t.Fatalf("expected chat and code present, got %v", types)
	}
}
