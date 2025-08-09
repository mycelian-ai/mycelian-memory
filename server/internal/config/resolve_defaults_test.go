package config

import (
	"os"
	"testing"
)

func unsetBuildEnv() {
	os.Unsetenv("MEMORY_BACKEND_BUILD_TARGET")
	os.Unsetenv("MEMORY_BACKEND_DB_DRIVER")
	os.Unsetenv("MEMORY_BACKEND_VECTOR_STORE")
}

func TestResolveDefaultsCloudDev(t *testing.T) {
	unsetBuildEnv()
	os.Setenv("MEMORY_BACKEND_BUILD_TARGET", "cloud-dev")
	defer unsetBuildEnv()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.DBDriver != "postgres" || cfg.VectorStore != "waviate" {
		t.Fatalf("unexpected mapping: %s %s", cfg.DBDriver, cfg.VectorStore)
	}
}

func TestResolveDefaultsOverride(t *testing.T) {
	unsetBuildEnv()
	os.Setenv("MEMORY_BACKEND_BUILD_TARGET", "local")
	os.Setenv("MEMORY_BACKEND_DB_DRIVER", "postgres")
	defer unsetBuildEnv()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.DBDriver != "postgres" {
		t.Fatalf("override failed, got %s", cfg.DBDriver)
	}
}

func TestResolveDefaultsLocal(t *testing.T) {
	unsetBuildEnv()
	os.Setenv("MEMORY_BACKEND_BUILD_TARGET", "local")
	os.Unsetenv("MEMORY_BACKEND_DB_DRIVER")
	os.Unsetenv("MEMORY_BACKEND_VECTOR_STORE")
	defer unsetBuildEnv()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.DBDriver != "postgres" || cfg.VectorStore != "waviate" {
		t.Fatalf("unexpected mapping for local: %s %s", cfg.DBDriver, cfg.VectorStore)
	}
}
