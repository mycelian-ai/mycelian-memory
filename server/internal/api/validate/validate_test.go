package validate

import (
	"strings"
	"testing"
)

func TestCreateUser_InvalidEmail(t *testing.T) {
	if err := CreateUser("alice", "bad email", nil); err == nil {
		t.Fatalf("expected error for invalid email")
	}
}

// No deep type check for tag values yet – only object type enforced.

func TestIsJSONObject(t *testing.T) {
	if err := IsJSONObject([]int{1}); err == nil {
		t.Fatalf("expected error for array")
	}
}

func TestContextFragments(t *testing.T) {
	bad := map[string]interface{}{"dri": 123}
	if err := ContextFragments(bad); err == nil {
		t.Fatalf("expected type error for context")
	}
	empty := map[string]interface{}{"dri": ""}
	if err := ContextFragments(empty); err == nil {
		t.Fatalf("expected empty fragment error")
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid title",
			title:       "Valid-Title123",
			expectError: false,
		},
		{
			name:        "empty title",
			title:       "",
			expectError: true,
			errorMsg:    "title is required",
		},
		{
			name:        "title too long",
			title:       strings.Repeat("a", 51),
			expectError: true,
			errorMsg:    "title exceeds 50 characters",
		},
		{
			name:        "title with spaces",
			title:       "Title With Spaces",
			expectError: true,
			errorMsg:    "title contains invalid characters; allowed letters, digits, hyphen",
		},
		{
			name:        "title with special characters",
			title:       "Title@Special!",
			expectError: true,
			errorMsg:    "title contains invalid characters; allowed letters, digits, hyphen",
		},
		{
			name:        "title with underscore",
			title:       "Title_With_Underscore",
			expectError: true,
			errorMsg:    "title contains invalid characters; allowed letters, digits, hyphen",
		},
		{
			name:        "title with apostrophe",
			title:       "Title's",
			expectError: true,
			errorMsg:    "title contains invalid characters; allowed letters, digits, hyphen",
		},
		{
			name:        "valid title with hyphens",
			title:       "Valid-Title-With-Hyphens",
			expectError: false,
		},
		{
			name:        "valid title with numbers",
			title:       "Title123",
			expectError: false,
		},
		{
			name:        "title at max length",
			title:       strings.Repeat("a", 50),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Title(tt.title)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for title '%s'", tt.title)
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Fatalf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for valid title '%s': %v", tt.title, err)
				}
			}
		})
	}
}

func TestCreateMemory(t *testing.T) {
	tests := []struct {
		name        string
		memoryType  string
		title       string
		description *string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid memory",
			memoryType:  "conversation",
			title:       "Valid-Title",
			description: stringPtr("Valid description"),
			expectError: false,
		},
		{
			name:        "empty memory type",
			memoryType:  "",
			title:       "Valid-Title",
			description: nil,
			expectError: true,
			errorMsg:    "memoryType is required",
		},
		{
			name:        "empty title",
			memoryType:  "conversation",
			title:       "",
			description: nil,
			expectError: true,
			errorMsg:    "title is required",
		},
		{
			name:        "title too long",
			memoryType:  "conversation",
			title:       strings.Repeat("a", 51),
			description: nil,
			expectError: true,
			errorMsg:    "title exceeds 50 characters",
		},
		{
			name:        "invalid title characters",
			memoryType:  "conversation",
			title:       "Invalid Title!",
			description: nil,
			expectError: true,
			errorMsg:    "title contains invalid characters; allowed letters, digits, hyphen",
		},
		{
			name:        "description too long",
			memoryType:  "conversation",
			title:       "Valid-Title",
			description: stringPtr(strings.Repeat("a", 501)),
			expectError: true,
			errorMsg:    "description exceeds 500 characters",
		},
		{
			name:        "description at max length",
			memoryType:  "conversation",
			title:       "Valid-Title",
			description: stringPtr(strings.Repeat("a", 500)),
			expectError: false,
		},
		{
			name:        "nil description",
			memoryType:  "conversation",
			title:       "Valid-Title",
			description: nil,
			expectError: false,
		},
		{
			name:        "empty description",
			memoryType:  "conversation",
			title:       "Valid-Title",
			description: stringPtr(""),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateMemory(tt.memoryType, tt.title, tt.description)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for test case '%s'", tt.name)
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Fatalf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for test case '%s': %v", tt.name, err)
				}
			}
		})
	}
}

func TestMaxLen(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       *string
		limit       int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil value",
			field:       "description",
			value:       nil,
			limit:       100,
			expectError: false,
		},
		{
			name:        "value within limit",
			field:       "description",
			value:       stringPtr("short"),
			limit:       100,
			expectError: false,
		},
		{
			name:        "value at limit",
			field:       "description",
			value:       stringPtr(strings.Repeat("a", 100)),
			limit:       100,
			expectError: false,
		},
		{
			name:        "value exceeds limit",
			field:       "description",
			value:       stringPtr(strings.Repeat("a", 101)),
			limit:       100,
			expectError: true,
			errorMsg:    "description exceeds 100 characters",
		},
		{
			name:        "empty string",
			field:       "description",
			value:       stringPtr(""),
			limit:       100,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxLen(tt.field, tt.value, tt.limit)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for test case '%s'", tt.name)
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Fatalf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for test case '%s': %v", tt.name, err)
				}
			}
		})
	}
}

func TestCreateMemoryEntry(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		summary     *string
		metadata    map[string]interface{}
		tags        map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid entry",
			raw:         "Valid raw entry content",
			summary:     stringPtr("Valid summary"),
			metadata:    map[string]interface{}{"key": "value"},
			tags:        map[string]interface{}{"tag": "value"},
			expectError: false,
		},
		{
			name:        "empty raw entry",
			raw:         "",
			summary:     stringPtr("Valid summary"),
			metadata:    nil,
			tags:        nil,
			expectError: true,
			errorMsg:    "rawEntry is required",
		},
		{
			name:        "raw entry at 9000 characters",
			raw:         strings.Repeat("a", 9000),
			summary:     stringPtr("Valid summary"),
			metadata:    nil,
			tags:        nil,
			expectError: false,
		},
		{
			name:        "raw entry exceeds 9000 characters",
			raw:         strings.Repeat("a", 9001),
			summary:     stringPtr("Valid summary"),
			metadata:    nil,
			tags:        nil,
			expectError: true,
			errorMsg:    "rawEntry exceeds 9000 characters",
		},
		{
			name:        "nil summary",
			raw:         "Valid raw entry",
			summary:     nil,
			metadata:    nil,
			tags:        nil,
			expectError: true,
			errorMsg:    "summary is required",
		},
		{
			name:        "empty summary",
			raw:         "Valid raw entry",
			summary:     stringPtr(""),
			metadata:    nil,
			tags:        nil,
			expectError: true,
			errorMsg:    "summary is required",
		},
		{
			name:        "valid metadata with complex values",
			raw:         "Valid raw entry",
			summary:     stringPtr("Valid summary"),
			metadata:    map[string]interface{}{"key": []int{1, 2, 3}},
			tags:        nil,
			expectError: false,
		},
		{
			name:        "valid tags with complex values",
			raw:         "Valid raw entry",
			summary:     stringPtr("Valid summary"),
			metadata:    nil,
			tags:        map[string]interface{}{"key": []int{1, 2, 3}},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateMemoryEntry(tt.raw, tt.summary, tt.metadata, tt.tags)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for test case '%s'", tt.name)
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Fatalf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for test case '%s': %v", tt.name, err)
				}
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
