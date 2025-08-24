// Package logger provides a configured zerolog logger.
package logger

import (
	"os"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog"
	zpkgerrors "github.com/rs/zerolog/pkgerrors"
)

// New returns a new zerolog.Logger configured for the application.
// Call sites should use .Stack() on error events to include stacks.
func New(serviceName string) zerolog.Logger {
	// Configure zerolog to work with github.com/pkg/errors:
	// - Automatically marshal pkg/errors stack traces when present
	// - Ensure a stack is present even for std errors when .Stack() is used
	zerolog.ErrorStackMarshaler = func(err error) interface{} {
		type stackTracer interface{ StackTrace() pkgerrors.StackTrace }
		if _, ok := err.(stackTracer); !ok {
			err = pkgerrors.WithStack(err)
		}
		return zpkgerrors.MarshalStack(err)
	}
	zerolog.ErrorMarshalFunc = func(err error) interface{} {
		// If the error already carries a pkg/errors stack, keep it.
		type stackTracer interface{ StackTrace() pkgerrors.StackTrace }
		if _, ok := err.(stackTracer); ok {
			return err
		}
		// Otherwise, attach a stack so downstream logging can render it.
		return pkgerrors.WithStack(err)
	}

	return zerolog.New(os.Stdout).With().
		Str("service", serviceName).
		Timestamp().
		Logger()
}
