package auth

import "errors"

var (
	// ErrMissingUserID is returned when user_id cannot be extracted from the request
	ErrMissingUserID = errors.New("user identification required")

	// ErrInvalidUserID is returned when user_id format is invalid
	ErrInvalidUserID = errors.New("invalid user identifier format")

	// ErrDevConfigError is returned when development config file cannot be accessed
	ErrDevConfigError = errors.New("cannot access development config file")

	// ErrDevUserInProduction is returned when development user IDs are used in production
	ErrDevUserInProduction = errors.New("development user IDs not allowed in production")
)
