package types

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/mycelian/mycelian-memory/client/internal/shardqueue"
)

// ------------------------------
// Shared Validation
// ------------------------------

// userIDRegex validates userID format
var UserIDRegex = regexp.MustCompile(`^[a-z0-9_]{1,20}$`)

// ValidateUserID validates user ID format
func ValidateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required")
	}
	if !UserIDRegex.MatchString(userID) {
		return fmt.Errorf("userId must be 1-20 characters containing only lowercase letters, digits, and underscore")
	}
	return nil
}

// titleRegex validates titles: ASCII letters, digits, spaces, hyphens (max 50 chars)
var titleRegex = regexp.MustCompile(`^[A-Za-z0-9\- ]+$`)

// ValidateTitle enforces title constraints used by vaults and memories.
// - 1-50 characters
// - ASCII letters (A-Z, a-z), digits (0-9), hyphens (-) only
func ValidateTitle(title string, fieldName string) error {
	if title == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if len(title) > 50 {
		return fmt.Errorf("%s exceeds 50 characters", fieldName)
	}
	if !titleRegex.MatchString(title) {
		return fmt.Errorf("%s contains invalid characters; allowed: letters, digits, spaces, hyphens", fieldName)
	}
	return nil
}

// ValidateIDPresent ensures a required identifier is non-empty.
func ValidateIDPresent(id string, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ------------------------------
// Shared Interfaces
// ------------------------------

// Executor interface for dependency injection (used by async operations)
type Executor interface {
	Submit(context.Context, string, shardqueue.Job) error
}

// HTTPClient interface for dependency injection
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ------------------------------
// Shared Errors
// ------------------------------

// ErrNotFound is returned when context snapshot is not found
var ErrNotFound = fmt.Errorf("context snapshot not found")
