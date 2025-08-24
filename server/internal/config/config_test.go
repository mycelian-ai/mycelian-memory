package config

import (
	"os"
	"testing"
)

func TestConfigLoad_EmbedDefaults(t *testing.T) {
	// clear env vars
	_ = os.Unsetenv("MEMORY_SERVER_EMBED_PROVIDER")
	_ = os.Unsetenv("MEMORY_SERVER_EMBED_MODEL")
	_ = os.Unsetenv("MEMORY_SERVER_SEARCH_ALPHA")

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.EmbedProvider != "ollama" || cfg.EmbedModel != "nomic-embed-text" || cfg.SearchAlpha != 0.6 {
		t.Fatalf("unexpected default embed config: %+v", cfg)
	}
}

func TestConfigLoad_EmbedEnvOverride(t *testing.T) {
	_ = os.Setenv("MEMORY_SERVER_EMBED_MODEL", "test-model")
	defer func() { _ = os.Unsetenv("MEMORY_SERVER_EMBED_MODEL") }()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.EmbedModel != "test-model" {
		t.Fatalf("embed model env override failed, got %s", cfg.EmbedModel)
	}
}

func TestConfigLoad_BootstrapTimeoutDefault(t *testing.T) {
	// clear env vars
	_ = os.Unsetenv("MEMORY_SERVER_BOOTSTRAP_TIMEOUT_SECONDS")

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.BootstrapTimeoutSeconds != 5 {
		t.Fatalf("unexpected default bootstrap timeout: %d", cfg.BootstrapTimeoutSeconds)
	}
}

func TestConfigLoad_BootstrapTimeoutEnvOverride(t *testing.T) {
	_ = os.Setenv("MEMORY_SERVER_BOOTSTRAP_TIMEOUT_SECONDS", "10")
	defer func() { _ = os.Unsetenv("MEMORY_SERVER_BOOTSTRAP_TIMEOUT_SECONDS") }()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.BootstrapTimeoutSeconds != 10 {
		t.Fatalf("bootstrap timeout env override failed, got %d", cfg.BootstrapTimeoutSeconds)
	}
}
