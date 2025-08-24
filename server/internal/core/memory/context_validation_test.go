package memory

import (
	"context"
	"testing"
)

func TestCreateMemoryContext_Validation(t *testing.T) {
	s := &Service{}
	// helper
	mkReq := func(payload string) CreateMemoryContextRequest {
		return CreateMemoryContextRequest{
			ActorID:  "u",
			MemoryID: "m",
			Context:  payload,
		}
	}

	cases := []struct {
		name string
		req  CreateMemoryContextRequest
	}{
		{"empty payload", mkReq(" ")},
	}

	for _, c := range cases {
		if _, err := s.CreateMemoryContext(context.TODO(), c.req); err == nil || !IsValidationError(err) {
			t.Fatalf("%s: expected validation error, got %v", c.name, err)
		}
	}
}
