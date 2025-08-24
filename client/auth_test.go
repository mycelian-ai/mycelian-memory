package client

import (
	"context"
	"testing"
)

func TestDevModeAuth(t *testing.T) {
	// Test dev mode authentication
	c, err := NewWithDevMode("http://localhost:11545")
	if err != nil {
		t.Fatalf("NewWithDevMode failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	t.Log("Testing authentication...")

	// Try an async operation that should trigger error logging
	_, err = c.AddEntry(context.Background(), "3ea7a8b3-93b4-44d1-b18e-f0a5b76ae31c", "2be61b26-e6f1-469d-b65d-ee4c5d5ee485", AddEntryRequest{
		RawEntry: "Test async auth",
		Summary:  "Testing",
	})
	if err != nil {
		t.Logf("AddEntry returned error: %v", err)
	} else {
		t.Log("AddEntry enqueued successfully")
	}

	// Wait for consistency - this should trigger the HTTP call and error logging
	t.Log("Awaiting consistency...")
	err = c.AwaitConsistency(context.Background(), "2be61b26-e6f1-469d-b65d-ee4c5d5ee485")
	if err != nil {
		t.Logf("AwaitConsistency failed: %v", err)
	} else {
		t.Log("Consistency achieved")
	}
}
