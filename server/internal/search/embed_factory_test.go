package search

import "testing"

func TestNewProvider_Ollama(t *testing.T) {
	emb, err := NewProvider("ollama", "mxbai-embed-large")
	if err != nil {
		t.Fatalf("expected provider, got error: %v", err)
	}
	if emb == nil {
		t.Fatalf("provider returned nil embedder")
	}
}
