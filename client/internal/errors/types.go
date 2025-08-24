// Package errors provides error classification for the client SDK.
// This enables different retry policies based on error recoverability.
package errors

import "fmt"

// ErrorCategory determines how errors should be handled by retry logic.
type ErrorCategory int

const (
	// Recoverable errors should be retried with exponential backoff.
	// Examples: 500 Internal Server Error, network timeouts, connection failures.
	Recoverable ErrorCategory = iota

	// Irrecoverable errors should fail immediately without retry.
	// Examples: 401 Unauthorized, 403 Forbidden, 400 Bad Request.
	Irrecoverable
)

// String returns a human-readable representation of the error category.
func (c ErrorCategory) String() string {
	switch c {
	case Recoverable:
		return "Recoverable"
	case Irrecoverable:
		return "Irrecoverable"
	default:
		return fmt.Sprintf("Unknown(%d)", int(c))
	}
}

// ClassifiedError wraps an error with categorization metadata for retry policies.
type ClassifiedError struct {
	Category   ErrorCategory
	StatusCode int    // HTTP status code (0 for non-HTTP errors)
	Body       string // Response body for debugging
	Underlying error  // The original error
}

// Error implements the error interface.
func (e *ClassifiedError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("[%s] HTTP %d: %v", e.Category, e.StatusCode, e.Underlying)
	}
	return fmt.Sprintf("[%s] %v", e.Category, e.Underlying)
}

// Unwrap returns the underlying error for error chain compatibility.
func (e *ClassifiedError) Unwrap() error {
	return e.Underlying
}

// IsIrrecoverable returns true if the error should not be retried.
func IsIrrecoverable(err error) bool {
	if classified, ok := err.(*ClassifiedError); ok {
		return classified.Category == Irrecoverable
	}
	return false
}
