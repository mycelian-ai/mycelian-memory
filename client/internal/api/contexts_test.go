package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockExec provided by mock_executor_provider_test.go

func TestPutContext_EnqueuesAndCallsHTTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	exec := &mockExec{}
	ack, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", "hello")
	if err != nil {
		t.Fatalf("PutContext error: %v", err)
	}
	if ack == nil || ack.MemoryID != "m1" {
		t.Fatalf("unexpected ack: %+v", ack)
	}
	if exec.n != 1 {
		t.Fatalf("expected one submit, got %d", exec.n)
	}
}

func TestGetContext_NotFoundMapsToErr(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := GetLatestContext(context.Background(), srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected ErrNotFound")
	}
}

func TestContexts_NonOKStatuses(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			w.WriteHeader(http.StatusBadRequest)
		case http.MethodGet:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()
	exec := &mockExec{}
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", "hello"); err == nil {
		t.Fatal("expected error for PutContext non-201")
	}
	if _, err := GetLatestContext(context.Background(), srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected error for GetLatestContext non-OK non-404")
	}
}

func TestPutContext_SubmitError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	exec := &failingExec{}
	_, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", "hello")
	if err == nil {
		t.Fatal("expected submit error from executor")
	}
}

func TestPutContext_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &mockExec{}
	if _, err := PutContext(ctx, exec, srv.Client(), srv.URL, "v1", "m1", "hello"); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestGetContext_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := GetLatestContext(ctx, srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected context canceled error for GetLatestContext")
	}
}

func TestGetContext_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := GetLatestContext(context.Background(), hc, "http://example.com", "v1", "m1"); err == nil {
		t.Fatal("expected http Do error for GetLatestContext")
	}
}

func TestPutContext_InvalidIDs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &mockExec{}
	// empty vaultId
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "", "m1", "hello"); err == nil {
		t.Fatal("expected validation error for empty vaultId")
	}
	// empty memoryId
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "", "hello"); err == nil {
		t.Fatal("expected validation error for empty memoryId")
	}
}
