package api

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestSearchRequestValidateDefaults(t *testing.T) {
	req := SearchRequest{UserID: "u1", MemoryID: "m1", Query: " test "}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Query != "test" || req.TopK != 10 {
		t.Fatalf("defaults not applied correctly: %+v", req)
	}
}

func TestSearchRequestValidateError(t *testing.T) {
	req := SearchRequest{UserID: "u1", MemoryID: "m1", Query: "   "}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected validation error for empty query")
	}
}

func TestDecodeSearchRequest(t *testing.T) {
	body := bytes.NewBufferString(`{"userId":"u1","memoryId":"m1","query":"foo","topK":5}`)
	r := httptest.NewRequest("POST", "/v0/search", body)
	sr, err := decodeSearchRequest(nil, r)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sr.Query != "foo" || sr.TopK != 5 {
		t.Fatalf("unexpected result: %+v", sr)
	}
}
