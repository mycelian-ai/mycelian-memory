package memory

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCreateMemoryContext_Validation(t *testing.T) {
	s := &Service{}
	// helper
	mkReq := func(payload string) CreateMemoryContextRequest {
		return CreateMemoryContextRequest{
			ActorID:  "u",
			MemoryID: "m",
			Context:  json.RawMessage(payload),
		}
	}

	cases := []struct {
		name string
		req  CreateMemoryContextRequest
	}{
		{"empty payload", mkReq(" ")},
		{"non string fragment", mkReq(`{"frag":123}`)},
		{"empty string fragment", mkReq(`{"frag":"  "}`)},
	}

	for _, c := range cases {
		if _, err := s.CreateMemoryContext(context.TODO(), c.req); err == nil || !IsValidationError(err) {
			t.Fatalf("%s: expected validation error, got %v", c.name, err)
		}
	}
}
