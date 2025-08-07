package client

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

var (
	// userIDRegex validates user IDs: 1-20 characters, lowercase letters, digits, underscore only
	userIDRegex = regexp.MustCompile(`^[a-z0-9_]{1,20}$`)

	// titleRegex validates titles: ASCII letters, digits, hyphens only (max 50 chars)
	titleRegex = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)
)

// ValidateUserID validates user ID according to API specification:
// - 1-20 characters
// - lowercase letters (a-z), digits (0-9), underscore (_) only
func ValidateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required")
	}
	if !userIDRegex.MatchString(userID) {
		return fmt.Errorf("userId must be 1-20 characters containing only lowercase letters, digits, and underscore")
	}
	return nil
}

// ValidateUUID validates that a string is a valid UUID format
// Used for vault IDs, memory IDs, entry IDs, context IDs
func ValidateUUID(id, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%s must be a valid UUID format", fieldName)
	}
	return nil
}

// ValidateTitle validates titles according to API specification:
// - 1-50 characters
// - ASCII letters (A-Z, a-z), digits (0-9), hyphens (-) only
func ValidateTitle(title, fieldName string) error {
	if title == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if len(title) > 50 {
		return fmt.Errorf("%s exceeds 50 characters", fieldName)
	}
	if !titleRegex.MatchString(title) {
		return fmt.Errorf("%s contains invalid characters; allowed: letters, digits, hyphens", fieldName)
	}
	return nil
}

// ValidateDescription validates optional description fields
func ValidateDescription(description string) error {
	if len(description) > 500 {
		return fmt.Errorf("description exceeds 500 characters")
	}
	return nil
}

// ValidateMemoryType validates memory type field
func ValidateMemoryType(memoryType string) error {
	if memoryType == "" {
		return fmt.Errorf("memoryType is required")
	}
	// Add specific memory type validation if needed
	return nil
}
