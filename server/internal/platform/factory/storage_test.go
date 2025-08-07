package factory

import (
	"testing"

	"github.com/mycelian/mycelian-memory/server/internal/config"
)

func TestNewStorage_SQLite(t *testing.T) {
	cfg := &config.Config{DBDriver: "sqlite", BuildTarget: "local"}
	// Ensure defaults so ResolveDefaults not needed
	st, err := NewStorage(cfg)
	if err != nil {
		t.Fatalf("NewStorage returned error for sqlite: %v", err)
	}
	if st == nil {
		t.Fatalf("Expected storage instance, got nil")
	}
}

func TestNewStorage_Unsupported(t *testing.T) {
	cfg := &config.Config{DBDriver: "spanner-pg"}
	_, err := NewStorage(cfg)
	if err == nil {
		t.Fatalf("Expected error for unimplemented spanner-pg adapter")
	}
}
