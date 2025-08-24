package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunContextPut_Smoke(t *testing.T) {
	vaultFlag = "v1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v0/vaults/v1/memories/m1/contexts") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Context map[string]interface{} `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Context["foo"] != "bar" {
			t.Fatalf("unexpected context payload: %+v", body.Context)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	var sb strings.Builder
	if err := runContextPut(srv.URL, "m1", `{"foo":"bar"}`, &sb); err != nil {
		t.Fatalf("runContextPut: %v", err)
	}
	if !strings.Contains(sb.String(), "foo") {
		t.Fatalf("unexpected output: %s", sb.String())
	}
}

func TestRunContextGet_Smoke(t *testing.T) {
	payload := `{"userId":"u1","vaultId":"v1","memoryId":"m1","contextId":"c1","context":{"foo":"bar"}}`
	vaultFlag = "v1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v0/vaults/v1/memories/m1/contexts") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	var sb strings.Builder
	if err := runContextGet(srv.URL, "m1", &sb); err != nil {
		t.Fatalf("runContextGet: %v", err)
	}
	if !strings.Contains(sb.String(), "foo") {
		t.Fatalf("unexpected output: %s", sb.String())
	}
}
