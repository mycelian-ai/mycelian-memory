package api

import (
	"os"
	"strings"
	"testing"
)

// TestAPIDocumentationContainsSearch ensures the search endpoint documentation exists.
func TestAPIDocumentationContainsSearch(t *testing.T) {
	data, err := os.ReadFile("../../../docs/server/api-documentation.md")
	if err != nil {
		t.Fatalf("read api doc: %v", err)
	}
	if !strings.Contains(string(data), "Search API") {
		t.Fatalf("docs/server/api-documentation.md missing 'Search API' section")
	}
}
