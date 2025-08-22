package errors

import "fmt"

// ClassifyHTTPError determines whether an HTTP error should be retried.
// This implements best practices for HTTP error handling:
// - 4xx client errors (except 429) are irrecoverable
// - 5xx server errors are recoverable 
// - Network-level errors are recoverable
func ClassifyHTTPError(statusCode int, body string, underlyingErr error) *ClassifiedError {
	category := getHTTPErrorCategory(statusCode)
	
	return &ClassifiedError{
		Category:   category,
		StatusCode: statusCode,
		Body:       body,
		Underlying: underlyingErr,
	}
}

// getHTTPErrorCategory maps HTTP status codes to error categories.
func getHTTPErrorCategory(statusCode int) ErrorCategory {
	switch {
	case statusCode >= 400 && statusCode < 500:
		// Client errors - generally irrecoverable
		switch statusCode {
		case 408: // Request Timeout - can retry
			return Recoverable
		case 429: // Too Many Requests - should retry with backoff
			return Recoverable
		default:
			// 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, etc.
			return Irrecoverable
		}
	case statusCode >= 500 && statusCode < 600:
		// Server errors - generally recoverable
		return Recoverable
	default:
		// Unexpected status codes - be conservative and retry
		return Recoverable
	}
}

// NewHTTPError creates a classified error for HTTP failures.
// This is a convenience function for API layer usage.
func NewHTTPError(statusCode int, body string, operation string) *ClassifiedError {
	underlyingErr := fmt.Errorf("%s failed: HTTP %d", operation, statusCode)
	return ClassifyHTTPError(statusCode, body, underlyingErr)
}

// NewNetworkError creates a classified error for network-level failures.
// Network errors are always recoverable as they may be transient.
func NewNetworkError(operation string, err error) *ClassifiedError {
	return &ClassifiedError{
		Category:   Recoverable,
		StatusCode: 0, // No HTTP status for network errors
		Body:       "",
		Underlying: fmt.Errorf("%s network error: %w", operation, err),
	}
}
