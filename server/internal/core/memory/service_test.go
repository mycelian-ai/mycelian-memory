package memory

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateCreateMemoryRequest_LengthLimits(t *testing.T) {
	s := &Service{}

	// Helper to create a valid base request
	validReq := func() CreateMemoryRequest {
		return CreateMemoryRequest{
			UserID:     "u",
			VaultID:    mustParseUUID("123e4567-e89b-12d3-a456-426614174000"),
			MemoryType: "PROJECT",
			Title:      "valid-title",
		}
	}

	// Test title length limit (50 characters)
	req := validReq()
	req.Title = strings.Repeat("a", 51)
	err := s.validateCreateMemoryRequest(req)
	if err == nil {
		t.Fatalf("expected error for long title")
	}
	if !strings.Contains(err.Error(), "title exceeds 50 characters") {
		t.Fatalf("expected title length error, got: %v", err)
	}

	// Test description length limit (500 characters)
	req = validReq()
	desc := strings.Repeat("b", 501)
	req.Description = &desc
	err = s.validateCreateMemoryRequest(req)
	if err == nil {
		t.Fatalf("expected error for long description")
	}
	if !strings.Contains(err.Error(), "description exceeds 500 characters") {
		t.Fatalf("expected description length error, got: %v", err)
	}

	// Test valid lengths
	req = validReq()
	req.Title = strings.Repeat("a", 50)
	validDesc := strings.Repeat("b", 500)
	req.Description = &validDesc
	err = s.validateCreateMemoryRequest(req)
	if err != nil {
		t.Fatalf("unexpected error for valid lengths: %v", err)
	}
}

// Helper function to parse UUID for tests
func mustParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
