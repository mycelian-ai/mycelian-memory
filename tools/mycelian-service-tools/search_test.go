package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunSearch_Smoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if auth := r.Header.Get("Authorization"); auth != "Bearer LOCAL_DEV_MODE_NOT_FOR_PRODUCTION" {
			t.Fatalf("unexpected auth header: %s", auth)
		}

		var req struct {
			MemoryID string `json:"memoryId"`
			Query    string `json:"query"`
			TopKE    int    `json:"top_ke"`
			TopKC    int    `json:"top_kc"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.MemoryID != "m1" || req.Query != "hello" || req.TopKE != 3 || req.TopKC != 2 {
			t.Fatalf("unexpected payload: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"entries":[],"count":0,"latestContext":"{}","latestContextTimestamp":"2025-01-04T12:00:00Z","contexts":[]}`))
	}))
	defer srv.Close()

	var sb strings.Builder
	if err := runSearch(srv.URL, "u1", "m1", "hello", 3, &sb); err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if !strings.Contains(sb.String(), "\"entries\"") {
		t.Fatalf("unexpected output: %s", sb.String())
	}
}
