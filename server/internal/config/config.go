package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Environment represents different deployment environments
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvTesting     Environment = "testing"
	EnvProduction  Environment = "production"
)

// Config holds the configuration for the memory service
// Environment variables are automatically parsed from MEMORY_BACKEND_ prefix
type Config struct {
	// Build target selects high-level environment: local, cloud-dev, cloud
	BuildTarget string `envconfig:"BUILD_TARGET" default:"cloud-dev"`

	// Derived or override drivers
	DBDriver    string `envconfig:"DB_DRIVER" default:"auto"`
	VectorStore string `envconfig:"VECTOR_STORE" default:"auto"`

	Environment  Environment `envconfig:"ENVIRONMENT" default:"development"`
	GCPProjectID string      `envconfig:"GCP_PROJECT_ID" default:"artful-guru-459003-k4"`

	// HTTP Configuration
	HTTPPort int `envconfig:"HTTP_PORT" default:"8080"`

	// gRPC Configuration
	GRPCPort int `envconfig:"GRPC_PORT" default:"9090"`

	// Spanner removed

	// Postgres Configuration
	PostgresDSN string `envconfig:"POSTGRES_DSN" default:""`

	// Embedding / Search Configuration
	EmbedProvider string  `envconfig:"EMBED_PROVIDER" default:"ollama"`
	EmbedModel    string  `envconfig:"EMBED_MODEL" default:"mxbai-embed-large"`
	SearchAlpha   float32 `envconfig:"SEARCH_ALPHA" default:"0.6"`

	// SQLite support removed
	// SQLitePath string `envconfig:"SQLITE_PATH" default:""`

	// Waviate (default for all targets)
	WaviateURL string `envconfig:"WAVIATE_URL" default:"weaviate:8080"`

	// Testing Configuration
	TestingUseEmulator  bool `envconfig:"TESTING_USE_EMULATOR" default:"true"`
	TestingTempDatabase bool `envconfig:"TESTING_TEMP_DATABASE" default:"true"`
	TestingParallel     bool `envconfig:"TESTING_PARALLEL" default:"true"`
}

// ResolveDefaults validates BuildTarget and derives DBDriver and VectorStore when set to "auto" or empty.
func (c *Config) ResolveDefaults() error {
	var defaultDB string

	switch c.BuildTarget {
	case "cloud-dev":
		defaultDB = "postgres"
	case "cloud":
		defaultDB = "postgres"
	case "local":
		defaultDB = "postgres"
	default:
		return fmt.Errorf("unsupported BUILD_TARGET: %s", c.BuildTarget)
	}

	if c.DBDriver == "" || c.DBDriver == "auto" {
		c.DBDriver = defaultDB
	}
	// Vector store is fixed to Waviate after M3 refactor.
	c.VectorStore = "waviate"

	// SQLite removed: no local file path derivation

	allowedDB := map[string]bool{"postgres": true}
	if !allowedDB[c.DBDriver] {
		return fmt.Errorf("unsupported DB_DRIVER: %s", c.DBDriver)
	}
	return nil
}

// New creates a new Config by parsing environment variables
// Environment variables should be prefixed with MEMORY_BACKEND_
// Example: MEMORY_BACKEND_GCP_PROJECT_ID, MEMORY_BACKEND_HTTP_PORT
func New() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("MEMORY_BACKEND", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	if err := cfg.ResolveDefaults(); err != nil {
		return nil, err
	}

	log.Info().
		Str("build_target", cfg.BuildTarget).
		Str("db_driver", cfg.DBDriver).
		Str("environment", string(cfg.Environment)).
		Str("project", cfg.GCPProjectID).
		Int("port", cfg.HTTPPort).
		// spanner removed
		Str("embed_provider", cfg.EmbedProvider).
		Str("embed_model", cfg.EmbedModel).
		Float32("search_alpha", cfg.SearchAlpha).
		// sqlite removed
		Str("postgres_dsn_present", func() string {
			if cfg.PostgresDSN != "" {
				return "true"
			}
			return "false"
		}()).
		Str("waviate_url", cfg.WaviateURL).
		Msg("Configuration loaded")

	return &cfg, nil
}

// NewForTesting creates a config specifically for testing
func NewForTesting() *Config {
	cfg := &Config{
		Environment:  EnvTesting,
		GCPProjectID: "test-project",
	}

	cfg.HTTPPort = 8080

	// spanner removed

	cfg.EmbedProvider = "ollama"
	cfg.EmbedModel = "mxbai-embed-large"
	cfg.SearchAlpha = 0.6
	cfg.WaviateURL = "localhost:8082"

	cfg.BuildTarget = "cloud-dev"
	cfg.DBDriver = "auto"
	cfg.VectorStore = "auto"

	cfg.TestingUseEmulator = true
	cfg.TestingTempDatabase = true
	cfg.TestingParallel = true

	return cfg
}

// IsTesting returns true if the environment is set to testing
func (c *Config) IsTesting() bool {
	return c.Environment == EnvTesting
}

// IsProduction returns true if the environment is set to production
func (c *Config) IsProduction() bool {
	return c.Environment == EnvProduction
}

// spanner helpers removed

// GetHTTPAddr returns the HTTP server address
func (c *Config) GetHTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

// GetGRPCAddr returns the gRPC server address
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}
