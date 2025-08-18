package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// mockExec provided by mock_executor_provider_test.go

func TestPutContext_EnqueuesAndCallsHTTP(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	exec := &mockExec{}
	ack, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", types.PutContextRequest{Context: map[string]any{"x": 1}})
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
	if _, err := GetContext(context.Background(), srv.Client(), srv.URL, "v1", "m1"); err == nil {
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
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", types.PutContextRequest{Context: map[string]any{"x": 1}}); err == nil {
		t.Fatal("expected error for PutContext non-201")
	}
	if _, err := GetContext(context.Background(), srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected error for GetContext non-OK non-404")
	}
}

func TestGetContext_DecodeError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv.Close()
	if _, err := GetContext(context.Background(), srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected decode error for GetContext")
	}
}

func TestGetContext_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"context": {"activeContext": "x"}}`))
	}))
	defer srv.Close()
	res, err := GetContext(context.Background(), srv.Client(), srv.URL, "v1", "m1")
	if err != nil || res == nil {
		t.Fatalf("GetContext success unexpected err=%v res=%+v", err, res)
	}
}

func TestPutContext_SubmitError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	var exec types.Executor = &failingExec{}
	_, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "m1", types.PutContextRequest{Context: map[string]any{"x": 1}})
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
	if _, err := PutContext(ctx, exec, srv.Client(), srv.URL, "v1", "m1", types.PutContextRequest{Context: map[string]any{"x": 1}}); err == nil {
		t.Fatal("expected context canceled error")
	}
}

func TestGetContext_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := GetContext(ctx, srv.Client(), srv.URL, "v1", "m1"); err == nil {
		t.Fatal("expected context canceled error for GetContext")
	}
}

func TestGetContext_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := GetContext(context.Background(), hc, "http://example.com", "v1", "m1"); err == nil {
		t.Fatal("expected http Do error for GetContext")
	}
}

func TestPutContext_InvalidIDs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	exec := &mockExec{}
	// empty vaultId
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "", "m1", types.PutContextRequest{Context: map[string]any{"x": 1}}); err == nil {
		t.Fatal("expected validation error for empty vaultId")
	}
	// empty memoryId
	if _, err := PutContext(context.Background(), exec, srv.Client(), srv.URL, "v1", "", types.PutContextRequest{Context: map[string]any{"x": 1}}); err == nil {
		t.Fatal("expected validation error for empty memoryId")
	}
}
