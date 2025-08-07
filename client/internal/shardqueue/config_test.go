package shardqueue

import (
	"testing"
)

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("SQ_SHARDS", "8")
	t.Setenv("SQ_QUEUE_SIZE", "256")
	t.Setenv("SQ_ENQUEUE_TIMEOUT", "250ms")
	t.Setenv("SQ_MAX_ATTEMPTS", "5")
	t.Setenv("SQ_BASE_BACKOFF", "200ms")
	t.Setenv("SQ_MAX_INTERVAL", "5s")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Shards != 8 || cfg.QueueSize != 256 {
		t.Fatalf("unexpected Shards/QueueSize: %+v", cfg)
	}
	if cfg.EnqueueTimeout.String() != "250ms" {
		t.Fatalf("unexpected EnqueueTimeout: %v", cfg.EnqueueTimeout)
	}
	if cfg.MaxAttempts != 5 {
		t.Fatalf("unexpected MaxAttempts: %v", cfg.MaxAttempts)
	}
	if cfg.BaseBackoff.String() != "200ms" || cfg.MaxInterval.String() != "5s" {
		t.Fatalf("unexpected backoff settings: base=%v max=%v", cfg.BaseBackoff, cfg.MaxInterval)
	}
}
