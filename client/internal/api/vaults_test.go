package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

func TestCreateVault_Success(t *testing.T) {
	t.Parallel()
	want := types.Vault{UserID: "test-user", VaultID: "v1", Title: "t"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	got, err := CreateVault(context.Background(), srv.Client(), srv.URL, types.CreateVaultRequest{Title: "t"})
	if err != nil || got == nil || got.VaultID != want.VaultID {
		t.Fatalf("CreateVault unexpected: got=%+v err=%v", got, err)
	}
}

func TestListVaults_Success(t *testing.T) {
	t.Parallel()
	resp := types.ListVaultsResponse{Vaults: []types.Vault{{VaultID: "v1"}}, Count: 1}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	got, err := ListVaults(context.Background(), srv.Client(), srv.URL)
	if err != nil || len(got) != 1 || got[0].VaultID != "v1" {
		t.Fatalf("ListVaults unexpected: got=%+v err=%v", got, err)
	}
}

func TestGetVault_Success(t *testing.T) {
	t.Parallel()
	want := types.Vault{VaultID: "v1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	got, err := GetVault(context.Background(), srv.Client(), srv.URL, "v1")
	if err != nil || got == nil || got.VaultID != "v1" {
		t.Fatalf("GetVault unexpected: got=%+v err=%v", got, err)
	}
}

func TestDeleteVault_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := DeleteVault(context.Background(), srv.Client(), srv.URL, "v1"); err != nil {
		t.Fatalf("DeleteVault error: %v", err)
	}
}

func TestVaults_InvalidUserID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := CreateVault(context.Background(), srv.Client(), srv.URL, types.CreateVaultRequest{Title: "t"}); err == nil {
		t.Fatal("expected HTTP error for CreateVault")
	}
	if _, err := ListVaults(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Fatal("expected HTTP error for ListVaults")
	}
	if _, err := GetVault(context.Background(), srv.Client(), srv.URL, "v1"); err == nil {
		t.Fatal("expected HTTP error for GetVault")
	}
	if err := DeleteVault(context.Background(), srv.Client(), srv.URL, "v1"); err == nil {
		t.Fatal("expected HTTP error for DeleteVault")
	}
}

func TestVaults_NonOKStatuses(t *testing.T) {
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
	if _, err := CreateVault(context.Background(), srv.Client(), srv.URL, types.CreateVaultRequest{Title: "t"}); err == nil {
		t.Fatal("expected error for CreateVault non-201")
	}
	if _, err := ListVaults(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Fatal("expected error for ListVaults non-200")
	}
	if _, err := GetVault(context.Background(), srv.Client(), srv.URL, "v1"); err == nil {
		t.Fatal("expected error for GetVault non-200")
	}
	if err := DeleteVault(context.Background(), srv.Client(), srv.URL, "v1"); err == nil {
		t.Fatal("expected error for DeleteVault non-204")
	}
}

func TestGetVaultByTitle_NonOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := GetVaultByTitle(context.Background(), srv.Client(), srv.URL, "title"); err == nil {
		t.Fatal("expected error for GetVaultByTitle non-200")
	}
}

func TestVaults_DecodeErrors(t *testing.T) {
	t.Parallel()
	// CreateVault decode error
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv1.Close()
	if _, err := CreateVault(context.Background(), srv1.Client(), srv1.URL, types.CreateVaultRequest{Title: "t"}); err == nil {
		t.Fatal("expected decode error for CreateVault")
	}

	// ListVaults decode error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv2.Close()
	if _, err := ListVaults(context.Background(), srv2.Client(), srv2.URL); err == nil {
		t.Fatal("expected decode error for ListVaults")
	}

	// GetVault decode error
	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer srv3.Close()
	if _, err := GetVault(context.Background(), srv3.Client(), srv3.URL, "v1"); err == nil {
		t.Fatal("expected decode error for GetVault")
	}
}

func TestVaults_HTTPDoError(t *testing.T) {
	t.Parallel()
	hc := &http.Client{Transport: &errRT{}}
	if _, err := CreateVault(context.Background(), hc, "http://example.com", types.CreateVaultRequest{Title: "t"}); err == nil {
		t.Fatal("expected Do error for CreateVault")
	}
	if _, err := ListVaults(context.Background(), hc, "http://example.com"); err == nil {
		t.Fatal("expected Do error for ListVaults")
	}
	if _, err := GetVault(context.Background(), hc, "http://example.com", "v1"); err == nil {
		t.Fatal("expected Do error for GetVault")
	}
	if err := DeleteVault(context.Background(), hc, "http://example.com", "v1"); err == nil {
		t.Fatal("expected Do error for DeleteVault")
	}
}

func TestCreateVault_CtxCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dummy := httptest.NewServer(http.NotFoundHandler())
	defer dummy.Close()
	if _, err := CreateVault(ctx, dummy.Client(), dummy.URL, types.CreateVaultRequest{Title: "t"}); err == nil {
		t.Fatal("expected context canceled for CreateVault")
	}
}

func TestCreateVault_InvalidTitle(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	// too long title (over 50 chars)
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz" // 52
	if _, err := CreateVault(context.Background(), srv.Client(), srv.URL, types.CreateVaultRequest{Title: long}); err == nil {
		t.Fatal("expected validation error for long title")
	}
}
