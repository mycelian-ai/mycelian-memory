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
		var req struct {
			UserID   string `json:"userId"`
			MemoryID string `json:"memoryId"`
			Query    string `json:"query"`
			TopK     int    `json:"topK"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.UserID != "u1" || req.MemoryID != "m1" || req.Query != "hello" || req.TopK != 3 {
			t.Fatalf("unexpected payload: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"entries":[],"count":0}`))
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
