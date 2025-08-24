package memory

import (
	"errors"
	"fmt"
)

// ValidationError represents a validation error in the domain
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
	}
}

// IsValidationError checks if an error is a validation error (including wrapped errors)
func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}

// ConflictError represents a unique constraint or duplicate resource error
type ConflictError struct {
	Field   string
	Message string
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Field, e.Message)
}

// NewConflictError constructs ConflictError
func NewConflictError(field, message string) ConflictError {
	return ConflictError{Field: field, Message: message}
}

// IsConflictError checks if error is ConflictError
func IsConflictError(err error) bool {
	var ce ConflictError
	return errors.As(err, &ce)
}

// NotFoundError represents missing related resource (e.g., parent row not found)
type NotFoundError struct {
	Field   string
	Message string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("not found %s: %s", e.Field, e.Message)
}

// NewNotFoundError constructs NotFoundError
func NewNotFoundError(field, message string) NotFoundError {
	return NotFoundError{Field: field, Message: message}
}

// IsNotFoundError checks if error is NotFoundError
func IsNotFoundError(err error) bool {
	var ne NotFoundError
	return errors.As(err, &ne)
}
