package client

import (
	"strings"
	"testing"
)

func TestValidateUserID(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
		errMsg  string
	}{
		{"empty userID", "", true, "userId is required"},
		{"valid userID lowercase", "user123", false, ""},
		{"valid userID with underscore", "user_123", false, ""},
		{"valid userID digits only", "123456", false, ""},
		{"valid userID max length", "a1234567890123456789", false, ""}, // 20 chars
		{"too long userID", "a12345678901234567890", true, "userId must be 1-20 characters"},
		{"uppercase not allowed", "User123", true, "userId must be 1-20 characters"},
		{"special chars not allowed", "user-123", true, "userId must be 1-20 characters"},
		{"spaces not allowed", "user 123", true, "userId must be 1-20 characters"},
		{"valid single char", "a", false, ""},
		{"valid underscore only", "_", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserID(tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateUserID() expected error for %q", tt.userID)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateUserID() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUserID() unexpected error for %q: %v", tt.userID, err)
				}
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{"empty UUID", "", "vault-id", true, "vault-id is required"},
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", "vault-id", false, ""},
		{"valid UUID v1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", "memory-id", false, ""},
		{"invalid UUID format", "not-a-uuid", "vault-id", true, "vault-id must be a valid UUID format"},
		{"invalid UUID short", "550e8400-e29b-41d4", "vault-id", true, "vault-id must be a valid UUID format"},
		{"invalid UUID characters", "550e8400-g29b-41d4-a716-446655440000", "memory-id", true, "memory-id must be a valid UUID format"},
		{"UUID without hyphens", "550e8400e29b41d4a716446655440000", "vault-id", false, ""}, // Go UUID parser accepts this format
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.id, tt.fieldName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateUUID() expected error for %q", tt.id)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateUUID() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUUID() unexpected error for %q: %v", tt.id, err)
				}
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{"empty title", "", "title", true, "title is required"},
		{"valid title letters", "ProjectTitle", "title", false, ""},
		{"valid title with digits", "Project123", "title", false, ""},
		{"valid title with hyphens", "Project-Title-123", "title", false, ""},
		{"max length title", strings.Repeat("a", 50), "title", false, ""},
		{"too long title", strings.Repeat("a", 51), "title", true, "title exceeds 50 characters"},
		{"invalid chars underscore", "Project_Title", "title", true, "title contains invalid characters"},
		{"invalid chars space", "Project Title", "title", true, "title contains invalid characters"},
		{"invalid chars special", "Project@Title", "title", true, "title contains invalid characters"},
		{"single char valid", "A", "title", false, ""},
		{"single hyphen valid", "-", "title", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title, tt.fieldName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTitle() expected error for %q", tt.title)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateTitle() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTitle() unexpected error for %q: %v", tt.title, err)
				}
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
		errMsg      string
	}{
		{"empty description", "", false, ""},
		{"valid short description", "A short description", false, ""},
		{"max length description", strings.Repeat("a", 500), false, ""},
		{"too long description", strings.Repeat("a", 501), true, "description exceeds 500 characters"},
		{"unicode characters", "Description with émojis 🎉", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.description)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDescription() expected error for %q", tt.description)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDescription() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDescription() unexpected error for %q: %v", tt.description, err)
				}
			}
		})
	}
}

func TestValidateMemoryType(t *testing.T) {
	tests := []struct {
		name       string
		memoryType string
		wantErr    bool
		errMsg     string
	}{
		{"empty memory type", "", true, "memoryType is required"},
		{"valid memory type", "conversation", false, ""},
		{"valid memory type uppercase", "CONVERSATION", false, ""},
		{"valid memory type mixed", "Project-Type", false, ""},
		{"single char", "A", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMemoryType(tt.memoryType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateMemoryType() expected error for %q", tt.memoryType)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMemoryType() error = %v, want to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateMemoryType() unexpected error for %q: %v", tt.memoryType, err)
				}
			}
		})
	}
}

// Test for deprecated function
func TestRequireUserID(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{"empty userID", "", true},
		{"valid userID", "test_user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserID(tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateUserID() expected error for %q", tt.userID)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUserID() unexpected error for %q: %v", tt.userID, err)
				}
			}
		})
	}
}
