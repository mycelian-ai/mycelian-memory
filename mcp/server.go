package mcp

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mycelian/mycelian-memory/client"
	"github.com/mycelian/mycelian-memory/mcp/internal/handlers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Configuration holds all settings for the MCP server
type config struct {
	MemoryServiceURL string
	ContextDataDir   string
	LogLevel         zerolog.Level
	ServerName       string
	ServerVersion    string
	ShutdownTimeout  time.Duration
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}

// loadConfig loads configuration from environment variables and flags
func loadConfig() *config {
	cfg := &config{
		// Default values
		MemoryServiceURL: getEnvOrDefault("MEMORY_SERVICE_URL", "http://localhost:11545"),
		ContextDataDir:   getEnvOrDefault("CONTEXT_DATA_DIR", "./data/context"),
		ServerName:       getEnvOrDefault("MCP_SERVER_NAME", "mycelian-mcp-server"),
		ServerVersion:    getEnvOrDefault("MCP_SERVER_VERSION", "0.2.0"),
		ShutdownTimeout:  parseDurationOrDefault("SHUTDOWN_TIMEOUT", "10s"),
		HTTPReadTimeout:  parseDurationOrDefault("HTTP_READ_TIMEOUT", "5s"),
		HTTPWriteTimeout: parseDurationOrDefault("HTTP_WRITE_TIMEOUT", "10s"),
		HTTPIdleTimeout:  parseDurationOrDefault("HTTP_IDLE_TIMEOUT", "120s"),
	}

	// Parse log level from environment
	cfg.LogLevel = parseLogLevel(getEnvOrDefault("LOG_LEVEL", "info"))

	// Command line flags (will override env vars)
	var rawLogLevel string
	flag.StringVar(&cfg.MemoryServiceURL, "memory-service-url", cfg.MemoryServiceURL, "Base URL of the Mycelian Memory Service")
	flag.StringVar(&cfg.ContextDataDir, "context-data-dir", cfg.ContextDataDir, "Filesystem directory where context docs are stored")
	flag.StringVar(&rawLogLevel, "log-level", cfg.LogLevel.String(), "Log level: debug|info|warn|error")
	flag.Parse()

	// Override log level from flag if provided
	if rawLogLevel != "" {
		cfg.LogLevel = parseLogLevel(rawLogLevel)
	}

	return cfg
}

// initLogger initializes the logger with the configured level
func (c *config) initLogger() {
	zerolog.SetGlobalLevel(c.LogLevel)
	log.Logger = log.With().Caller().Logger()
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDurationOrDefault(envKey, defaultValue string) time.Duration {
	if value := os.Getenv(envKey); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	d, _ := time.ParseDuration(defaultValue)
	return d
}

func parseLogLevel(levelStr string) zerolog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

type toolRegisterer interface {
	RegisterTools(s *server.MCPServer) error
}

func registerHandler(s *server.MCPServer, handler toolRegisterer, name string) {
	if err := handler.RegisterTools(s); err != nil {
		log.Fatal().Err(err).Msgf("Failed to register %s tools", name)
	}
}

// RunMCPServer starts the MCP server with the given configuration
func RunMCPServer() error {
	// Load configuration and initialize dependencies
	cfg := loadConfig()
	cfg.initLogger()

	// Initialize the new Client SDK
	log.Info().Str("memory_service_url", cfg.MemoryServiceURL).Msg("Creating client with dev mode")
	mycelianClient, err := client.NewWithDevMode(cfg.MemoryServiceURL)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to create client")
		return err
	}
	log.Info().Msg("Client created successfully")

	// Create a new MCP server
	s := server.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		server.WithToolCapabilities(true),
		// Advertise empty resources & prompts so the host client stops returning
		// -32601 for resources/list and prompts/list.
		server.WithResourceCapabilities(true, true), // subscribe=true, listChanged=true
		server.WithPromptCapabilities(true),         // listChanged=true
	)

	// Initialize and register handlers
	registerHandler(s, handlers.NewMemoryHandler(mycelianClient), "memory")
	registerHandler(s, handlers.NewEntryHandler(mycelianClient), "entry")
	registerHandler(s, handlers.NewSearchHandler(mycelianClient), "search")
	registerHandler(s, handlers.NewPromptsHandler(mycelianClient), "prompts")
	registerHandler(s, handlers.NewVaultHandler(mycelianClient), "vault")
	registerHandler(s, handlers.NewContextHandler(mycelianClient), "context")
	registerHandler(s, handlers.NewConsistencyHandler(mycelianClient), "consistency")

	// Auto-detect transport method
	if shouldUseStdio() {
		// Stdio transport (for Claude Desktop, launched processes)
		log.Info().Msg("Starting Mycelian MCP server (stdio transport)")

		// Serve over stdio using the correct API
		if err := server.ServeStdio(s); err != nil {
			log.Fatal().Err(err).Msg("Stdio server error")
		}
	} else {
		// HTTP transport (for manual/Docker startup)
		log.Info().Msg("Starting Mycelian MCP server (Streamable HTTP) on :11546")

		// Set up graceful shutdown handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Create shutdown coordination channel
		shutdownComplete := make(chan struct{})

		// Serve via Streamable HTTP transport on :11546
		streamSrv := server.NewStreamableHTTPServer(
			s,
			server.WithEndpointPath("/mcp"),
			server.WithHeartbeatInterval(30*time.Second),
		)

		srv := &http.Server{
			Addr:         ":11546",
			Handler:      streamSrv,
			ReadTimeout:  cfg.HTTPReadTimeout, // Keep short for request parsing
			WriteTimeout: 0,                   // No deadline - required for SSE streaming
			IdleTimeout:  cfg.HTTPIdleTimeout, // Keep for after requests finish
		}

		// Graceful shutdown handler
		go func() {
			defer close(shutdownComplete)

			sig := <-sigChan
			log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

			// Create shutdown context with timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()

			// Shutdown HTTP server first
			log.Info().Msg("Shutting down HTTP server...")
			if err := srv.Shutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("Error during HTTP server shutdown")
			} else {
				log.Info().Msg("HTTP server shutdown complete")
			}

			// Shutdown streamable MCP server
			log.Info().Msg("Shutting down MCP streamable server...")
			if err := streamSrv.Shutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("Error during MCP server shutdown")
			} else {
				log.Info().Msg("MCP server shutdown complete")
			}

			// Shutdown Mycelian client
			log.Info().Msg("Shutting down Mycelian client...")
			if err := mycelianClient.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing Mycelian client")
			} else {
				log.Info().Msg("Mycelian client shutdown complete")
			}
		}()

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}

		// Wait for graceful shutdown to complete
		<-shutdownComplete
		log.Info().Msg("MCP server shutdown complete")
	}

	return nil
}

// shouldUseStdio determines whether to use stdio transport based on environment
func shouldUseStdio() bool {
	// Force stdio mode with environment variable
	if os.Getenv("MCP_STDIO") == "true" {
		return true
	}

	// Force HTTP mode with environment variable
	if os.Getenv("MCP_HTTP") == "true" {
		return false
	}

	// Auto-detect: Use stdio if stdin is not a terminal (launched by another process)
	if fileInfo, err := os.Stdin.Stat(); err == nil {
		return (fileInfo.Mode() & os.ModeCharDevice) == 0
	}

	// Default to HTTP if detection fails
	return false
}
