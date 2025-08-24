package config

import (
	"os"
	"testing"
)

func unsetBuildEnv() {
	_ = os.Unsetenv("MEMORY_SERVER_BUILD_TARGET")
	_ = os.Unsetenv("MEMORY_SERVER_DB_DRIVER")
}

func TestResolveDefaultsCloudDev(t *testing.T) {
	unsetBuildEnv()
	_ = os.Setenv("MEMORY_SERVER_BUILD_TARGET", "cloud-dev")
	defer unsetBuildEnv()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.DBDriver != "postgres" {
		t.Fatalf("unexpected mapping: %s", cfg.DBDriver)
	}
}

func TestResolveDefaultsOverride(t *testing.T) {
	unsetBuildEnv()
	_ = os.Setenv("MEMORY_SERVER_BUILD_TARGET", "local")
	_ = os.Setenv("MEMORY_SERVER_DB_DRIVER", "postgres")
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
	_ = os.Setenv("MEMORY_SERVER_BUILD_TARGET", "local")
	_ = os.Unsetenv("MEMORY_SERVER_DB_DRIVER")
	defer unsetBuildEnv()

	cfg, err := New()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if cfg.DBDriver != "postgres" {
		t.Fatalf("unexpected mapping for local: %s", cfg.DBDriver)
	}
}
