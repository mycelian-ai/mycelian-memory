package storage

import "errors"

// Standard storage layer errors that abstract underlying implementation details.
// These errors should be used by all storage implementations to provide consistent
// error semantics across the application.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("storage: resource not found")

	// ErrConflict indicates a unique constraint violation or duplicate resource.
	ErrConflict = errors.New("storage: resource already exists")

	// ErrNotImplemented indicates the operation is not yet supported.
	ErrNotImplemented = errors.New("storage: operation not implemented")
)
