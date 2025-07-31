package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds application configuration.
type Config struct {
	MemoryServiceURL string
	LogLevel         zerolog.Level
	ContextDataDir   string
}

// Load loads application configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		MemoryServiceURL: getEnvOrDefault("MEMORY_SERVICE_URL", "http://localhost:8080"),
		LogLevel:         getLogLevel(),
		ContextDataDir:   getEnvOrDefault("CONTEXT_DATA_DIR", "./data/context"),
	}
	return cfg
}

// Init initializes all application dependencies.
func (c *Config) Init() {
	InitLogger()
	SetLogLevel(c.LogLevel)

	log.Info().
		Str("memory_service_url", c.MemoryServiceURL).
		Str("context_data_dir", c.ContextDataDir).
		Str("log_level", c.LogLevel.String()).
		Msg("Application configuration loaded")
}

// getEnvOrDefault returns environment variable value or default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getLogLevel parses log level from environment or returns default.
func getLogLevel() zerolog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug", "DEBUG":
		return zerolog.DebugLevel
	case "info", "INFO":
		return zerolog.InfoLevel
	case "warn", "WARN":
		return zerolog.WarnLevel
	case "error", "ERROR":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
