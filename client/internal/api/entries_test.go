package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// Use mockExec from contexts_test.go; extend behavior via helper methods if needed.

func TestAddEntry_EnqueuesAndCallsHTTP(t *testing.T) {
	t.Parallel()
	// Backend returns 201
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	exec := &mockExec{}
	ack, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "user_1", "v1", "m1", types.AddEntryRequest{RawEntry: "hi"})
	if err != nil {
		t.Fatalf("AddEntry error: %v", err)
	}
	if ack == nil || ack.MemoryID != "m1" || ack.Status != "enqueued" {
		t.Fatalf("unexpected ack: %+v", ack)
	}
	if len(exec.calls) != 1 || exec.calls[0] != "m1" {
		t.Fatalf("expected one Submit call for shard m1, got %+v", exec.calls)
	}
}

func TestDeleteEntry_SyncCallsHTTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := DeleteEntry(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1", "e1"); err != nil {
		t.Fatalf("DeleteEntry error: %v", err)
	}
}

func TestEntries_InvalidUserID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &mockExec{}
	if _, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "BAD ID!", "v1", "m1", types.AddEntryRequest{RawEntry: "hi"}); err == nil {
		t.Fatal("expected validation error for AddEntry")
	}
	if _, err := ListEntries(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1", "m1", nil); err == nil {
		t.Fatal("expected validation error for ListEntries")
	}
	if err := DeleteEntry(context.Background(), srv.Client(), srv.URL, "BAD ID!", "v1", "m1", "e1"); err == nil {
		t.Fatal("expected validation error for DeleteEntry")
	}
}

func TestEntries_NonOKStatuses(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
		case http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
		case http.MethodDelete:
			w.WriteHeader(http.StatusConflict)
		}
	}))
	defer srv.Close()
	exec := &mockExec{}
	if _, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "user_1", "v1", "m1", types.AddEntryRequest{RawEntry: "hi"}); err == nil {
		t.Fatal("expected error for AddEntry non-201")
	}
	if _, err := ListEntries(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1", nil); err == nil {
		t.Fatal("expected error for ListEntries non-200")
	}
	if err := DeleteEntry(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1", "e1"); err == nil {
		t.Fatal("expected error for DeleteEntry non-204")
	}
}

func TestListEntries_DecodeError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv.Close()
	if _, err := ListEntries(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1", nil); err == nil {
		t.Fatal("expected decode error for ListEntries")
	}
}

// errRT returns an error from RoundTrip to simulate network errors.
func TestListEntries_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := ListEntries(context.Background(), hc, "http://example.com", "user_1", "v1", "m1", nil); err == nil {
		t.Fatal("expected http Do error for ListEntries")
	}
}

func TestListEntries_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dummy := httptest.NewServer(http.NotFoundHandler())
	defer dummy.Close()
	if _, err := ListEntries(ctx, dummy.Client(), dummy.URL, "user_1", "v1", "m1", nil); err == nil {
		t.Fatal("expected context canceled for ListEntries")
	}
}

func TestListEntries_WithParams(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"entries":[],"count":0}`))
	}))
	defer srv.Close()
	if _, err := ListEntries(context.Background(), srv.Client(), srv.URL, "user_1", "v1", "m1", map[string]string{"limit": "2", "offset": "1"}); err != nil {
		t.Fatalf("ListEntries with params: %v", err)
	}
}

func TestAddEntry_SubmitError(t *testing.T) {
	t.Parallel()
	// Server won't be called because Submit fails
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &failingExec{}
	if _, err := AddEntry(context.Background(), exec, srv.Client(), srv.URL, "user_1", "v1", "m1", types.AddEntryRequest{RawEntry: "hi"}); err == nil {
		t.Fatal("expected submit error for AddEntry")
	}
}

func TestAddEntry_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &mockExec{}
	if _, err := AddEntry(ctx, exec, srv.Client(), srv.URL, "user_1", "v1", "m1", types.AddEntryRequest{RawEntry: "hi"}); err == nil {
		t.Fatal("expected context canceled for AddEntry")
	}
}

func TestDeleteEntry_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if err := DeleteEntry(context.Background(), hc, "http://example.com", "user_1", "v1", "m1", "e1"); err == nil {
		t.Fatal("expected http Do error for DeleteEntry")
	}
}

func TestDeleteEntry_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if err := DeleteEntry(ctx, srv.Client(), srv.URL, "user_1", "v1", "m1", "e1"); err == nil {
		t.Fatal("expected context canceled for DeleteEntry")
	}
}
