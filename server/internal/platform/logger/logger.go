// Package logger provides a configured zerolog logger.
package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// New returns a new zerolog.Logger configured for the application.
func New(serviceName string) zerolog.Logger {
	return zerolog.New(os.Stdout).With().
		Str("service", serviceName).
		Timestamp().
		Logger()
}
