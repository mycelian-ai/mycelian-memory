package config

import (
	"os"
	"testing"
)

func TestConfigLoad_EmbedDefaults(t *testing.T) {
	// clear env vars
	_ = os.Unsetenv("MEMORY_BACKEND_EMBED_PROVIDER")
	_ = os.Unsetenv("MEMORY_BACKEND_EMBED_MODEL")
	_ = os.Unsetenv("MEMORY_BACKEND_SEARCH_ALPHA")

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.EmbedProvider != "ollama" || cfg.EmbedModel != "mxbai-embed-large" || cfg.SearchAlpha != 0.6 {
		t.Fatalf("unexpected default embed config: %+v", cfg)
	}
}

func TestConfigLoad_EmbedEnvOverride(t *testing.T) {
	_ = os.Setenv("MEMORY_BACKEND_EMBED_MODEL", "test-model")
	defer func() { _ = os.Unsetenv("MEMORY_BACKEND_EMBED_MODEL") }()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.EmbedModel != "test-model" {
		t.Fatalf("embed model env override failed, got %s", cfg.EmbedModel)
	}
}
