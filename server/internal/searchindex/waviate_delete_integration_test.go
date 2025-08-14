package searchindex_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// TestWaviateDeleteSuite validates delete operations on the search index implementation.
// Requires a running Waviate instance (set WAVIATE_URL to host:port). Skipped otherwise.
func TestWaviateDeleteSuite(t *testing.T) {
	host := os.Getenv("WAVIATE_URL")
	if host == "" {
		t.Skip("WAVIATE_URL not set; skipping search index delete suite")
	}

	if err := searchindex.BootstrapWaviate(context.Background(), host); err != nil {
		t.Fatalf("bootstrap waviate: %v", err)
	}

	idx, err := searchindex.NewWaviateNativeIndex(host)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Best-effort deletes should no-op without error
	if err := idx.DeleteEntry(ctx, "test-user", "non-existent-entry"); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if err := idx.DeleteContext(ctx, "test-user", "non-existent-context"); err != nil {
		t.Fatalf("DeleteContext: %v", err)
	}
	if err := idx.DeleteMemory(ctx, "test-user", "non-existent-memory"); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if err := idx.DeleteVault(ctx, "test-user", "non-existent-vault"); err != nil {
		t.Fatalf("DeleteVault: %v", err)
	}
}
