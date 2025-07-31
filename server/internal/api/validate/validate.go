package validate

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var emailRx = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// titleRx allows ASCII letters, digits, single spaces, hyphen, underscore and apostrophe.
// We deliberately keep it simple to meet the "English + space" requirement.
var titleRx = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)

// UserID must be lowercase letters, digits, underscore, 1-20 chars
var userIdRx = regexp.MustCompile(`^[a-z0-9_]{1,20}$`)

// Title validates that a title string conforms to our rules:
// - 1–50 bytes
// - ASCII letters/digits/space/hyphen/underscore/apostrophe only
// - No leading/trailing space
// - No consecutive spaces
// Returns an error describing the first violated rule.
func Title(v string) error {
	if v == "" {
		return fmt.Errorf("title is required")
	}

	if len(v) > 50 {
		return fmt.Errorf("title exceeds 50 characters")
	}

	if !titleRx.MatchString(v) {
		return fmt.Errorf("title contains invalid characters; allowed letters, digits, hyphen")
	}

	return nil
}

func Email(v string) error {
	if v == "" {
		return fmt.Errorf("email is required")
	}
	if len(v) > 320 || !emailRx.MatchString(v) {
		return fmt.Errorf("invalid email")
	}
	return nil
}

func NonEmpty(field, v string) error {
	if v == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func MaxLen(field string, v *string, limit int) error {
	if v == nil {
		return nil
	}
	if len(*v) > limit {
		return fmt.Errorf("%s exceeds %d characters", field, limit)
	}
	return nil
}

func IsJSONObject(val interface{}) error {
	switch v := val.(type) {
	case map[string]interface{}:
		return nil
	case json.RawMessage:
		var m map[string]interface{}
		if err := json.Unmarshal(v, &m); err == nil {
			return nil
		}
	}
	return fmt.Errorf("must be JSON object")
}

// -------- Request specific helpers ----------

// CreateUser validates input for creating a new user. UserID is mandatory.
func CreateUser(userId, email string, displayName *string) error {
	if userId == "" {
		return fmt.Errorf("userId is required")
	}
	if !userIdRx.MatchString(userId) {
		return fmt.Errorf("userId must match %s", userIdRx.String())
	}
	if err := Email(email); err != nil {
		return err
	}
	if err := MaxLen("displayName", displayName, 100); err != nil {
		return err
	}
	return nil
}

func CreateMemory(memoryType, title string, description *string) error {
	if err := NonEmpty("memoryType", memoryType); err != nil {
		return err
	}
	// Title validation (length, charset, spacing)
	if err := Title(title); err != nil {
		return err
	}
	if err := MaxLen("description", description, 500); err != nil {
		return err
	}
	return nil
}

func CreateMemoryEntry(raw string, summary *string, metadata, tags map[string]interface{}) error {
	if err := NonEmpty("rawEntry", raw); err != nil {
		return err
	}
	if len(raw) > 9000 {
		return fmt.Errorf("rawEntry exceeds 9000 characters")
	}
	if summary == nil || *summary == "" {
		return fmt.Errorf("summary is required")
	}
	if metadata != nil {
		if err := IsJSONObject(metadata); err != nil {
			return fmt.Errorf("metadata %w", err)
		}
	}
	if tags != nil {
		if err := IsJSONObject(tags); err != nil {
			return fmt.Errorf("tags %w", err)
		}
	}
	return nil
}

func ContextFragments(ctx map[string]interface{}) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	for k, v := range ctx {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("context fragment %s must be string", k)
		}
		if s == "" {
			return fmt.Errorf("context fragment %s must not be empty", k)
		}
	}
	return nil
}
