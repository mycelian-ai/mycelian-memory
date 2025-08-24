package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	// Arrange
	req := types.CreateUserRequest{UserID: "user_1", Email: "u@example.com"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v0/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		// echo minimal user
		var got types.CreateUserRequest
		_ = json.NewDecoder(r.Body).Decode(&got)
		b, _ := json.Marshal(types.User{ID: got.UserID, Email: got.Email})
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	// Act
	u, err := CreateUser(context.Background(), srv.Client(), srv.URL, req)

	// Assert
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if u == nil || u.ID != req.UserID || u.Email != req.Email {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestCreateUser_InputValidation(t *testing.T) {
	t.Parallel()
	// Missing userId should be rejected before HTTP call
	req := types.CreateUserRequest{UserID: "", Email: "u@example.com"}
	dummy := httptest.NewServer(http.NotFoundHandler())
	defer dummy.Close()
	if _, err := CreateUser(context.Background(), dummy.Client(), dummy.URL, req); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateUser_Non201(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	_, err := CreateUser(context.Background(), srv.Client(), srv.URL, types.CreateUserRequest{UserID: "user_1"})
	if err == nil {
		t.Fatal("expected error for non-201 status")
	}
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	want := types.User{ID: "user_1", Email: "u@example.com"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		b, _ := json.Marshal(want)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	got, err := GetUser(context.Background(), srv.Client(), srv.URL, want.ID)
	if err != nil {
		t.Fatalf("GetUser error: %v", err)
	}
	if got == nil || got.ID != want.ID || got.Email != want.Email {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := GetUser(context.Background(), srv.Client(), srv.URL, "BAD ID!"); err == nil {
		t.Fatal("expected validation error for userID")
	}
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
		_, _ = w.Write([]byte{})
	}))
	defer srv.Close()
	if err := DeleteUser(context.Background(), srv.Client(), srv.URL, "user_1"); err != nil {
		t.Fatalf("DeleteUser error: %v", err)
	}
}

func TestDeleteUser_Non204(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()
	if err := DeleteUser(context.Background(), srv.Client(), srv.URL, "user_1"); err == nil {
		t.Fatal("expected error for non-204 status")
	}
}

func TestUsers_DecodeError(t *testing.T) {
	t.Parallel()
	// Return malformed JSON to trigger decode error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv.Close()
	if _, err := GetUser(context.Background(), srv.Client(), srv.URL, "user_1"); err == nil {
		t.Fatal("expected decode error from GetUser")
	}
}

func TestGetUser_NonOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := GetUser(context.Background(), srv.Client(), srv.URL, "user_1"); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestCreateUser_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dummy := httptest.NewServer(http.NotFoundHandler())
	defer dummy.Close()
	if _, err := CreateUser(ctx, dummy.Client(), dummy.URL, types.CreateUserRequest{UserID: "user_1", Email: "u@example.com"}); err == nil {
		t.Fatal("expected context canceled for CreateUser")
	}
}

func TestUsers_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := CreateUser(context.Background(), hc, "http://example.com", types.CreateUserRequest{UserID: "u", Email: "e@example.com"}); err == nil {
		t.Fatal("expected Do error for CreateUser")
	}
	if _, err := GetUser(context.Background(), hc, "http://example.com", "u"); err == nil {
		t.Fatal("expected Do error for GetUser")
	}
	if err := DeleteUser(context.Background(), hc, "http://example.com", "u"); err == nil {
		t.Fatal("expected Do error for DeleteUser")
	}
}
