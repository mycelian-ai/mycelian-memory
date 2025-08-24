package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/auth"
	"github.com/mycelian/mycelian-memory/server/internal/model"
)

type mockEmbedder struct {
	calls int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.calls++
	return []float32{1.0, 2.0}, nil
}

type mockSearch struct {
	calls int
	empty bool
}

func (m *mockSearch) Search(ctx context.Context, uid, mid, q string, v []float32, k int, a float32) ([]model.SearchHit, error) {
	m.calls++
	if m.empty {
		return []model.SearchHit{}, nil
	}
	return []model.SearchHit{{EntryID: "e1", Summary: "s", Score: 0.9}}, nil
}

func (m *mockSearch) LatestContext(ctx context.Context, uid, mid string) (string, time.Time, error) {
	return "ctx", time.Now(), nil
}

func (m *mockSearch) BestContext(ctx context.Context, uid, mid, q string, v []float32, a float32) (string, time.Time, float64, error) {
	return "bestctx", time.Now(), 0.9, nil
}

// new interface methods (no-ops for tests)
func (m *mockSearch) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	return nil
}

func (m *mockSearch) UpsertContext(ctx context.Context, ctxID string, vec []float32, payload map[string]interface{}) error {
	return nil
}

func (m *mockSearch) DeleteEntry(ctx context.Context, userID, entryID string) error     { return nil }
func (m *mockSearch) DeleteContext(ctx context.Context, userID, contextID string) error { return nil }
func (m *mockSearch) DeleteMemory(ctx context.Context, userID, memoryID string) error   { return nil }
func (m *mockSearch) DeleteVault(ctx context.Context, userID, vaultID string) error     { return nil }

type mockAuthorizer struct{}

func (m *mockAuthorizer) Authorize(ctx context.Context, apiKey, operation, resource string) (*auth.ActorInfo, error) {
	return &auth.ActorInfo{
		ActorID:     "test-user",
		ProjectID:   "test-project",
		OrgID:       "test-org",
		KeyType:     "admin",
		KeyName:     "Test Key",
		Permissions: []string{"*"},
	}, nil
}

func TestHandleSearch_EmbedsOnce(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hello","topK":3}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()

	h.HandleSearch(w, req)

	if emb.calls != 1 {
		t.Fatalf("expected 1 embed call, got %d", emb.calls)
	}
	if srch.calls != 1 {
		t.Fatalf("expected 1 search call")
	}
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Removed legacy hybrid builder test; current handler uses native index directly

func TestHandleSearch_ResponseMapping(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hi"}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200")
	}
	var resp struct {
		Entries       []model.SearchHit `json:"entries"`
		Count         int               `json:"count"`
		LatestContext string            `json:"latestContext"`
		BestContext   string            `json:"bestContext"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 || len(resp.Entries) != 1 || resp.Entries[0].EntryID != "e1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.LatestContext == "" || resp.BestContext == "" {
		t.Fatalf("expected context fields")
	}
}

func TestHandleSearch_NoResults(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{empty: true}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hi"}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 0 {
		t.Fatalf("expected count 0, got %d", resp.Count)
	}
}
