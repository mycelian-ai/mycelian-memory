package respond

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// helper to decode ErrorResponse
func decodeErrorResponse(t *testing.T, rr *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var er ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &er); err != nil {
		t.Fatalf("failed to unmarshal error response: %v; body=%s", err, rr.Body.String())
	}
	return er
}

func TestHandleError_NotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleError(rr, storage.ErrNotFound, "resource not found")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNotFound)
	}
	er := decodeErrorResponse(t, rr)
	if er.Code != http.StatusNotFound || er.Error != http.StatusText(http.StatusNotFound) {
		t.Fatalf("unexpected error response: %+v", er)
	}
	if er.Message != "resource not found" {
		t.Fatalf("message: got %q, want %q", er.Message, "resource not found")
	}
}

func TestHandleError_Conflict(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleError(rr, storage.ErrConflict, "conflict")
	if rr.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusConflict)
	}
	er := decodeErrorResponse(t, rr)
	if er.Code != http.StatusConflict || er.Error != http.StatusText(http.StatusConflict) {
		t.Fatalf("unexpected error response: %+v", er)
	}
	if er.Message != "conflict" {
		t.Fatalf("message: got %q, want %q", er.Message, "conflict")
	}
}

func TestHandleError_NotImplemented(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleError(rr, storage.ErrNotImplemented, "not implemented")
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNotImplemented)
	}
	er := decodeErrorResponse(t, rr)
	if er.Code != http.StatusNotImplemented || er.Error != http.StatusText(http.StatusNotImplemented) {
		t.Fatalf("unexpected error response: %+v", er)
	}
	if er.Message != "not implemented" {
		t.Fatalf("message: got %q, want %q", er.Message, "not implemented")
	}
}

func TestHandleError_Unknown(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleError(rr, errors.New("boom"), "something failed")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	er := decodeErrorResponse(t, rr)
	if er.Code != http.StatusInternalServerError || er.Error != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("unexpected error response: %+v", er)
	}
	// For unknown errors, implementation returns generic message to avoid leaking internals
	if er.Message != "internal server error" {
		t.Fatalf("message: got %q, want %q", er.Message, "internal server error")
	}
}

func TestHandleError_NilError(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleError(rr, nil, "ignored-default")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	er := decodeErrorResponse(t, rr)
	if er.Message != "unexpected nil error" {
		t.Fatalf("message: got %q, want %q", er.Message, "unexpected nil error")
	}
}
