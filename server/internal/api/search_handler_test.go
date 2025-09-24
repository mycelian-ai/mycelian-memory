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

func (m *mockSearch) Search(ctx context.Context, uid, mid, q string, v []float32, kE int, a float32, includeRawEntries bool) ([]model.SearchHit, error) {
	m.calls++
	if m.empty {
		return []model.SearchHit{}, nil
	}
	now := time.Now()
	return []model.SearchHit{{EntryID: "e1", Summary: "s", Score: 0.9, CreationTime: now, ConversationTime: now.Add(-time.Hour)}}, nil
}

func (m *mockSearch) LatestContext(ctx context.Context, uid, mid string) (string, time.Time, error) {
	return "ctx", time.Now(), nil
}

func (m *mockSearch) SearchContexts(ctx context.Context, uid, mid, q string, v []float32, kC int, a float32) ([]model.ContextHit, error) {
	out := make([]model.ContextHit, 0, kC)
	for i := 0; i < kC; i++ {
		out = append(out, model.ContextHit{Context: "ctx", Timestamp: time.Now(), Score: 0.8})
	}
	return out, nil
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

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hello","top_ke":2,"top_kc":1}`)
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

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hi","top_ke":5,"top_kc":2}`)
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
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 || len(resp.Entries) != 1 || resp.Entries[0].EntryID != "e1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.LatestContext == "" {
		t.Fatalf("expected latestContext to be present")
	}
}

func TestHandleSearch_ContextsArray_KCLimit(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hi","top_ke":5,"top_kc":1}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200")
	}
	var resp struct {
		Contexts               []map[string]any `json:"contexts"`
		LatestContext          string           `json:"latestContext"`
		LatestContextTimestamp string           `json:"latestContextTimestamp"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Contexts) == 0 || len(resp.Contexts) > 1 {
		t.Fatalf("expected contexts length 1, got %d", len(resp.Contexts))
	}
	if resp.LatestContext == "" {
		t.Fatalf("expected latestContext to be present")
	}
}

func TestHandleSearch_NoResults(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{empty: true}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"hi","top_ke":5,"top_kc":2}`)
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

func TestHandleSearch_ConversationTimeInResults(t *testing.T) {
	emb := &mockEmbedder{}
	srch := &mockSearch{}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"test","top_ke":1,"top_kc":1}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Entries []model.SearchHit `json:"entries"`
		Count   int               `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Count != 1 || len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", resp.Count)
	}

	entry := resp.Entries[0]
	if entry.ConversationTime.IsZero() {
		t.Fatalf("expected ConversationTime to be set in search result")
	}

	// Verify ConversationTime is different from CreationTime (as per mock)
	if entry.ConversationTime.Equal(entry.CreationTime) {
		t.Fatalf("expected ConversationTime to be different from CreationTime")
	}
}

func TestHandleSearch_MultipleEntriesWithVariedConversationTimes(t *testing.T) {
	emb := &mockEmbedder{}
	// Create mock that returns multiple entries with different conversation times
	srch := &mockSearchMultiple{}
	auth := &mockAuthorizer{}
	h, _ := NewSearchHandler(emb, srch, 0.6, auth)

	body := bytes.NewBufferString(`{"memoryId":"m1","query":"test","top_ke":5,"top_kc":1}`)
	req := httptest.NewRequest("POST", "/v0/search", body)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	h.HandleSearch(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Entries []model.SearchHit `json:"entries"`
		Count   int               `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Count != 3 {
		t.Fatalf("expected 3 entries, got %d", resp.Count)
	}

	// Verify each entry has unique ConversationTime
	seenTimes := make(map[time.Time]bool)
	for i, entry := range resp.Entries {
		if entry.ConversationTime.IsZero() {
			t.Errorf("entry %d has zero ConversationTime", i)
		}

		// Check for duplicates
		if seenTimes[entry.ConversationTime] {
			t.Errorf("duplicate ConversationTime found: %v", entry.ConversationTime)
		}
		seenTimes[entry.ConversationTime] = true

		// Verify ConversationTime is not in the future
		if entry.ConversationTime.After(time.Now().Add(time.Minute)) {
			t.Errorf("entry %d has future ConversationTime: %v", i, entry.ConversationTime)
		}
	}
}

// mockSearchMultiple returns multiple entries with varied conversation times
type mockSearchMultiple struct{}

func (m *mockSearchMultiple) Search(ctx context.Context, uid, mid, q string, v []float32, kE int, a float32, includeRawEntries bool) ([]model.SearchHit, error) {
	now := time.Now()
	return []model.SearchHit{
		{EntryID: "e1", Summary: "old entry", Score: 0.9, CreationTime: now.Add(-48*time.Hour), ConversationTime: now.Add(-72*time.Hour)},
		{EntryID: "e2", Summary: "recent entry", Score: 0.8, CreationTime: now.Add(-2*time.Hour), ConversationTime: now.Add(-3*time.Hour)},
		{EntryID: "e3", Summary: "current entry", Score: 0.7, CreationTime: now, ConversationTime: now},
	}, nil
}

func (m *mockSearchMultiple) LatestContext(ctx context.Context, uid, mid string) (string, time.Time, error) {
	return "ctx", time.Now(), nil
}

func (m *mockSearchMultiple) SearchContexts(ctx context.Context, uid, mid, q string, v []float32, kC int, a float32) ([]model.ContextHit, error) {
	return []model.ContextHit{{Context: "ctx", Timestamp: time.Now(), Score: 0.8}}, nil
}

func (m *mockSearchMultiple) UpsertEntry(ctx context.Context, entryID string, vec []float32, payload map[string]interface{}) error {
	return nil
}

func (m *mockSearchMultiple) UpsertContext(ctx context.Context, ctxID string, vec []float32, payload map[string]interface{}) error {
	return nil
}

func (m *mockSearchMultiple) DeleteEntry(ctx context.Context, userID, entryID string) error   { return nil }
func (m *mockSearchMultiple) DeleteContext(ctx context.Context, userID, contextID string) error { return nil }
func (m *mockSearchMultiple) DeleteMemory(ctx context.Context, userID, memoryID string) error  { return nil }
func (m *mockSearchMultiple) DeleteVault(ctx context.Context, userID, vaultID string) error    { return nil }
