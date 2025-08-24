package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAndDeleteEntries(t *testing.T) {
	vaultID, memID, entryID := "v1", "m1", "e1"
	var getCalled, deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			getCalled = true
			resp := ListEntriesResponse{Entries: []Entry{{ID: entryID}}, Count: 1}
			_ = json.NewEncoder(w).Encode(&resp)
			return
		}
		if r.Method == http.MethodDelete {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Fatalf("unexpected method %s", r.Method)
	}))
	defer srv.Close()

	c, err := NewWithDevMode(srv.URL)
	if err != nil {
		t.Fatalf("NewWithDevMode: %v", err)
	}
	ctx := context.Background()
	lr, err := c.ListEntries(ctx, vaultID, memID, map[string]string{"limit": "10"})
	if err != nil || lr.Count != 1 {
		t.Fatalf("ListEntries error: %v", err)
	}
	if !getCalled {
		t.Fatalf("GET not called")
	}
	if err := c.DeleteEntry(ctx, vaultID, memID, entryID); err != nil {
		t.Fatalf("DeleteEntry error: %v", err)
	}
	if !deleteCalled {
		t.Fatalf("DELETE not called")
	}
}
