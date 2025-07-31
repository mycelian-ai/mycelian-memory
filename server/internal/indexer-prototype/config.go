package indexer

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds runtime configuration for the indexer-prototype service.
// Values are taken in this precedence order:
//   1. Command-line flag
//   2. Environment variable
//   3. Default (hard-coded)
//
// This keeps early scaffolding simple without introducing another dependency.
// We can replace this with envconfig if needed later.
//
// Environment variable names are documented next to each field.
//
// Example CLI run:
//   go run ./cmd/indexer-prototype -waviate-url=http://localhost:8081 -interval=30s
//
// Example with env vars only:
//   export SPANNER_PROJECT_ID=test-project
//   export WAVIATE_URL=http://localhost:8081
//   go run ./cmd/indexer-prototype
//
// NOTE: All durations are parsable by time.ParseDuration (e.g. "10s", "5m").
//
//nolint:lll // long struct tags for clarity

type Config struct {
	// Underlying database driver name (spanner-pg | sqlite)
	DBDriver string // env: DB_DRIVER default: spanner-pg

	// Path to SQLite file when DB_DRIVER=sqlite
	SQLitePath string // env: SQLITE_PATH default: ~/.synapse-memory/memory.db

	// Spanner logical identifiers (required even for emulator)
	SpannerProjectID    string // env: SPANNER_PROJECT_ID   default: test-project
	SpannerInstanceID   string // env: SPANNER_INSTANCE_ID  default: test-instance
	SpannerDatabaseID   string // env: SPANNER_DATABASE_ID  default: test-database
	SpannerEmulatorHost string // env: SPANNER_EMULATOR_HOST default: localhost:9010

	// Waviate base URL, e.g. http://localhost:8081
	WaviateURL string // env: WAVIATE_URL  default: http://localhost:8081

	// Embedding provider selector ("ollama" | "openai")
	Provider string // env: EMBED_PROVIDER default: ollama

	// Embedding model name for the provider (e.g. mxbai-embed-large, nomic-embed-text)
	EmbedModel string // env: EMBED_MODEL default: mxbai-embed-large

	// Hybrid search alpha (0.0 – 1.0)
	SearchAlpha float64 // env: SEARCH_ALPHA default: 0.6

	// Interval between scans
	Interval time.Duration // env: INDEX_INTERVAL default: 1s

	// Log level (info, debug, warn, error)
	LogLevel string // env: LOG_LEVEL default: info

	// Run indexer loop only once (for smoke testing); if true the process exits after
	// a single embed/scan cycle. Env: INDEXER_ONCE default: false
	Once bool
}

// Load parses flags and env vars into a Config struct.
func Load() Config {
	var cfg Config

	// Defaults from env vars or hard-coded fallback
	cfg.SpannerProjectID = getEnv("SPANNER_PROJECT_ID", "test-project")
	cfg.SpannerInstanceID = getEnv("SPANNER_INSTANCE_ID", "test-instance")
	cfg.SpannerDatabaseID = getEnv("SPANNER_DATABASE_ID", "test-database")
	cfg.SpannerEmulatorHost = getEnv("SPANNER_EMULATOR_HOST", "localhost:9010")
	cfg.WaviateURL = getEnv("WAVIATE_URL", "http://localhost:8081")
	cfg.Provider = getEnv("EMBED_PROVIDER", "ollama")
	cfg.EmbedModel = getEnv("EMBED_MODEL", "mxbai-embed-large")
	cfg.SearchAlpha = mustParseFloat(getEnv("SEARCH_ALPHA", "0.6"))
	cfg.Interval = mustParseDuration(getEnv("INDEX_INTERVAL", "1s"))
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// Parse bool env manually (optional)
	if val := os.Getenv("INDEXER_ONCE"); val == "1" || val == "true" {
		cfg.Once = true
	}

	// Parse DBDriver and SQLitePath
	cfg.DBDriver = getEnv("DB_DRIVER", "spanner-pg")
	cfg.SQLitePath = getEnv("SQLITE_PATH", defaultSQLitePath())

	// Flags override env/defaults
	flag.StringVar(&cfg.SpannerProjectID, "spanner-project", cfg.SpannerProjectID, "Spanner project ID")
	flag.StringVar(&cfg.SpannerInstanceID, "spanner-instance", cfg.SpannerInstanceID, "Spanner instance ID")
	flag.StringVar(&cfg.SpannerDatabaseID, "spanner-database", cfg.SpannerDatabaseID, "Spanner database ID")
	flag.StringVar(&cfg.SpannerEmulatorHost, "spanner-emulator", cfg.SpannerEmulatorHost, "Spanner emulator host (host:port)")
	flag.StringVar(&cfg.WaviateURL, "waviate-url", cfg.WaviateURL, "Base URL for Waviate")
	flag.StringVar(&cfg.Provider, "provider", cfg.Provider, "Embedding provider (ollama|openai)")
	flag.StringVar(&cfg.EmbedModel, "embed-model", cfg.EmbedModel, "Embedding model name for provider")
	flag.DurationVar(&cfg.Interval, "interval", cfg.Interval, "Scan interval (e.g. 10s)")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug|info|warn|error)")
	flag.BoolVar(&cfg.Once, "once", cfg.Once, "Run a single cycle and exit (smoke test)")
	flag.Float64Var(&cfg.SearchAlpha, "search-alpha", cfg.SearchAlpha, "Hybrid search alpha")

	flag.Parse()

	return cfg
}

// helper functions
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustParseDuration(v string) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		// fall back to seconds if parse fails
		return 10 * time.Second
	}
	return d
}

func mustParseFloat(v string) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		// fall back to default value if parse fails
		return 0.6
	}
	return f
}

func defaultSQLitePath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".synapse-memory", "memory.db")
	}
	return "memory.db"
}
