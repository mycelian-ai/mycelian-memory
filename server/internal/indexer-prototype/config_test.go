package indexer

import (
	"flag"
	"os"
	"testing"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{os.Args[0]}
}

func TestLoadDefaults(t *testing.T) {
	resetFlags()
	os.Clearenv()

	cfg := Load()

	if cfg.EmbedModel != "mxbai-embed-large" {
		t.Fatalf("expected default embed model mxbai-embed-large, got %s", cfg.EmbedModel)
	}
	if cfg.Provider != "ollama" {
		t.Fatalf("expected default provider ollama, got %s", cfg.Provider)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	resetFlags()
	os.Setenv("EMBED_MODEL", "nomic-embed-text")
	os.Setenv("INDEXER_ONCE", "true")
	t.Cleanup(os.Clearenv)

	cfg := Load()

	if !cfg.Once {
		t.Fatalf("expected Once flag true from env var")
	}
	if cfg.EmbedModel != "nomic-embed-text" {
		t.Fatalf("expected embed model override, got %s", cfg.EmbedModel)
	}
}

func TestLoadDefaults_SearchAlpha(t *testing.T) {
	resetFlags()
	os.Unsetenv("SEARCH_ALPHA")
	cfg := Load()
	if cfg.SearchAlpha != 0.6 {
		t.Fatalf("expected default alpha 0.6, got %f", cfg.SearchAlpha)
	}
}

func TestEnvOverride_SearchAlpha(t *testing.T) {
	resetFlags()
	os.Setenv("SEARCH_ALPHA", "0.9")
	defer os.Unsetenv("SEARCH_ALPHA")
	cfg := Load()
	if cfg.SearchAlpha != 0.9 {
		t.Fatalf("env override failed, got %f", cfg.SearchAlpha)
	}
}
