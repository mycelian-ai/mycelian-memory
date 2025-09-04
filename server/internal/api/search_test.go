package api

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestSearchRequestValidateDefaults(t *testing.T) {
	req := SearchRequest{MemoryID: "m1", Query: " test "}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Query != "test" || req.TopKE == nil || *req.TopKE != 5 || req.TopKC == nil || *req.TopKC != 2 {
		t.Fatalf("defaults not applied correctly: topKE=%v, topKC=%v", req.TopKE, req.TopKC)
	}
}

func TestSearchRequestValidateError(t *testing.T) {
	req := SearchRequest{MemoryID: "m1", Query: "   "}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected validation error for empty query")
	}
}

func TestDecodeSearchRequest(t *testing.T) {
	body := bytes.NewBufferString(`{"memoryId":"m1","query":"foo","top_ke":3,"top_kc":1}`)
	r := httptest.NewRequest("POST", "/v0/search", body)
	sr, err := decodeSearchRequest(nil, r)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sr.Query != "foo" || sr.TopKE == nil || *sr.TopKE != 3 || sr.TopKC == nil || *sr.TopKC != 1 {
		t.Fatalf("unexpected result: topKE=%v, topKC=%v", sr.TopKE, sr.TopKC)
	}
}

func TestSearchRequestValidateRanges(t *testing.T) {
	tests := []struct {
		name    string
		topKE   int
		topKC   int
		wantErr bool
	}{
		{"valid max values", 10, 3, false},
		{"valid min values", 0, 1, false},
		{"topKE too high", 11, 2, true},
		{"topKE negative", -1, 2, true},
		{"topKC too high", 5, 4, true},
		{"topKC zero", 5, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SearchRequest{
				MemoryID: "m1",
				Query:    "test",
				TopKE:    &tt.topKE,
				TopKC:    &tt.topKC,
			}
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
