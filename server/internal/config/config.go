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
// Environment variables are automatically parsed from MEMORY_SERVER_ prefix
type Config struct {
	// Build target selects high-level environment: local, cloud-dev, cloud
	BuildTarget string `envconfig:"BUILD_TARGET" default:"cloud-dev"`

	// Derived or override drivers
	DBDriver string `envconfig:"DB_DRIVER" default:"auto"`

	Environment Environment `envconfig:"ENVIRONMENT" default:"development"`

	// Development Mode Configuration
	DevMode   bool   `envconfig:"DEV_MODE" default:"false"`
	DevUserID string `envconfig:"DEV_USER_ID" default:"mycelian-dev"`

	// HTTP Configuration
	HTTPPort int `envconfig:"HTTP_PORT" default:"11545"`

	// gRPC Configuration
	GRPCPort int `envconfig:"GRPC_PORT" default:"9090"`

	// Postgres Configuration
	PostgresDSN string `envconfig:"POSTGRES_DSN" default:""`

	// Embedding / Search Configuration
	EmbedProvider string  `envconfig:"EMBED_PROVIDER" default:"ollama"`
	EmbedModel    string  `envconfig:"EMBED_MODEL" default:"nomic-embed-text"`
	SearchAlpha   float32 `envconfig:"SEARCH_ALPHA" default:"0.6"`

	// Vector search index endpoint (provider-agnostic)
	SearchIndexURL string `envconfig:"SEARCH_INDEX_URL" default:""`

	// Health checker configuration
	HealthIntervalSeconds     int `envconfig:"HEALTH_INTERVAL_SECONDS" default:"30"`
	HealthProbeTimeoutSeconds int `envconfig:"HEALTH_PROBE_TIMEOUT_SECONDS" default:"2"`

	// Bootstrap timeout configuration (in seconds)
	BootstrapTimeoutSeconds int `envconfig:"BOOTSTRAP_TIMEOUT_SECONDS" default:"5"`

	// Testing Configuration
	TestingUseEmulator  bool `envconfig:"TESTING_USE_EMULATOR" default:"true"`
	TestingTempDatabase bool `envconfig:"TESTING_TEMP_DATABASE" default:"true"`
	TestingParallel     bool `envconfig:"TESTING_PARALLEL" default:"true"`

	// Context handling
	// Maximum allowed size in characters (Unicode code points) for a context document (0 disables limit)
	MaxContextChars int `envconfig:"MAX_CONTEXT_CHARS" default:"65536"`
}

// ResolveDefaults validates BuildTarget and derives DBDriver when set to "auto" or empty.
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

	allowedDB := map[string]bool{"postgres": true}
	if !allowedDB[c.DBDriver] {
		return fmt.Errorf("unsupported DB_DRIVER: %s", c.DBDriver)
	}
	return nil
}

// New creates a new Config by parsing environment variables
// Environment variables should be prefixed with MEMORY_SERVER_
// Example: MEMORY_SERVER_HTTP_PORT, MEMORY_SERVER_POSTGRES_DSN
func New() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("MEMORY_SERVER", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	if err := cfg.ResolveDefaults(); err != nil {
		return nil, err
	}

	log.Info().
		Str("build_target", cfg.BuildTarget).
		Str("db_driver", cfg.DBDriver).
		Str("environment", string(cfg.Environment)).
		Int("port", cfg.HTTPPort).
		Str("embed_provider", cfg.EmbedProvider).
		Str("embed_model", cfg.EmbedModel).
		Float32("search_alpha", cfg.SearchAlpha).
		Str("postgres_dsn_present", func() string {
			if cfg.PostgresDSN != "" {
				return "true"
			}
			return "false"
		}()).
		Str("search_index_url_present", func() string {
			if cfg.SearchIndexURL != "" {
				return "true"
			}
			return "false"
		}()).
		Msg("Configuration loaded")

	return &cfg, nil
}

// NewForTesting creates a config specifically for testing
func NewForTesting() *Config {
	cfg := &Config{
		Environment: EnvTesting,
	}

	cfg.HTTPPort = 11545

	cfg.EmbedProvider = "ollama"
	cfg.EmbedModel = "nomic-embed-text"
	cfg.SearchAlpha = 0.6
	cfg.SearchIndexURL = "localhost:11543"

	cfg.BuildTarget = "cloud-dev"
	cfg.DBDriver = "auto"

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

// GetHTTPAddr returns the HTTP server address
func (c *Config) GetHTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

// GetGRPCAddr returns the gRPC server address
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}

// IsDevMode returns true if development mode is enabled
func (c *Config) IsDevMode() bool {
	return c.DevMode
}
