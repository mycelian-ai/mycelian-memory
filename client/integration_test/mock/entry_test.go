package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	client "github.com/mycelian/mycelian-memory/client"
)

func TestClient_AddEntry(t *testing.T) {
	type resp struct {
		status int
		body   interface{}
	}

	tests := []struct {
		name        string
		serverRes   resp
		wantErr     bool
		cancelCtx   bool
		setTimeout  time.Duration
		serverDelay time.Duration
	}{
		{
			name: "201 created",
			serverRes: resp{
				status: http.StatusCreated,
				body:   map[string]string{"message": "Entry created successfully"},
			},
			wantErr: false,
		},
		{
			name:      "500 internal server error (enqueued successfully)",
			serverRes: resp{status: http.StatusInternalServerError, body: map[string]string{"error": "failure"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name:      "400 bad request (enqueued successfully)",
			serverRes: resp{status: http.StatusBadRequest, body: map[string]string{"error": "bad"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name:      "404 not found (enqueued successfully)",
			serverRes: resp{status: http.StatusNotFound, body: map[string]string{"error": "nf"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name:      "503 service unavailable (enqueued successfully)",
			serverRes: resp{status: http.StatusServiceUnavailable, body: map[string]string{"error": "svc"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name:      "413 payload too large (enqueued successfully)",
			serverRes: resp{status: http.StatusRequestEntityTooLarge, body: map[string]string{"error": "large"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name:      "429 rate limited (enqueued successfully)",
			serverRes: resp{status: http.StatusTooManyRequests, body: map[string]string{"error": "limit"}},
			wantErr:   false, // Job enqueues successfully, HTTP error happens later
		},
		{
			name: "context cancelled (pre-enqueue)",
			// We still configure a normal 201 response, but the request should
			// never reach the server because the context is already cancelled.
			serverRes: resp{status: http.StatusCreated, body: map[string]string{"message": "Entry created successfully"}},
			wantErr:   true,
			cancelCtx: true,
		},
		{
			name:        "deadline exceeded (enqueued successfully)",
			serverRes:   resp{status: http.StatusCreated, body: map[string]string{"message": "ok"}},
			wantErr:     false, // Job enqueues successfully, timeout happens later during execution
			setTimeout:  50 * time.Millisecond,
			serverDelay: 100 * time.Millisecond,
		},
		{
			name:      "invalid user id validation error (pre-enqueue)",
			serverRes: resp{status: http.StatusCreated, body: map[string]string{"message": "ok"}},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverDelay > 0 {
					time.Sleep(tt.serverDelay)
				}
				callCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverRes.status)
				_ = json.NewEncoder(w).Encode(tt.serverRes.body)
			}))
			defer srv.Close()

			c := client.New(srv.URL)
			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if tt.setTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.setTimeout)
				defer cancel()
			}

			userID := "user1"
			if tt.name == "invalid user id validation error (pre-enqueue)" {
				userID = "BAD ID!"
			}
			_, err := c.AddEntry(ctx, userID, "vlt-1", "mem-1", client.AddEntryRequest{RawEntry: "hello"})
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Ensure no HTTP request was sent when context was pre-cancelled.
			if tt.cancelCtx && callCount != 0 {
				t.Fatalf("expected 0 outbound requests, got %d", callCount)
			}
		})
	}
}

func TestClient_DeleteEntry_Success(t *testing.T) {
	t.Parallel()
	userID, vaultID, memID, entryID := "u1", "v1", "m1", "e1"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/"+userID+"/vaults/"+vaultID+"/memories/"+memID+"/entries/"+entryID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := client.New(srv.URL)
	t.Cleanup(func() { _ = c.Close() })
	if err := c.DeleteEntry(context.Background(), userID, vaultID, memID, entryID); err != nil {
		t.Fatalf("delete entry: %v", err)
	}
}

func TestClient_GetEntry_Success(t *testing.T) {
	t.Parallel()
	userID, vaultID, memID, entryID := "u1", "v1", "m1", "e1"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users/"+userID+"/vaults/"+vaultID+"/memories/"+memID+"/entries/"+entryID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entryId":      entryID,
			"userId":       userID,
			"vaultId":      vaultID,
			"memoryId":     memID,
			"creationTime": time.Now().UTC().Format(time.RFC3339),
			"rawEntry":     "hello",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := client.New(srv.URL)
	t.Cleanup(func() { _ = c.Close() })
	got, err := c.GetEntry(context.Background(), userID, vaultID, memID, entryID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if got == nil || got.ID != entryID || got.MemoryID != memID {
		t.Fatalf("unexpected entry: %+v", got)
	}
}
