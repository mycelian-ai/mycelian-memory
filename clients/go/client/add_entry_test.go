package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
			name:      "500 error (async, expect no client error)",
			serverRes: resp{status: http.StatusInternalServerError, body: map[string]string{"error": "failure"}},
			wantErr:   false,
		},
		{
			name:      "400 bad request (async)",
			serverRes: resp{status: http.StatusBadRequest, body: map[string]string{"error": "bad"}},
			wantErr:   false,
		},
		{
			name:      "404 not found (async)",
			serverRes: resp{status: http.StatusNotFound, body: map[string]string{"error": "nf"}},
			wantErr:   false,
		},
		{
			name:      "503 service unavailable (async)",
			serverRes: resp{status: http.StatusServiceUnavailable, body: map[string]string{"error": "svc"}},
			wantErr:   false,
		},
		{
			name:      "413 payload too large (async)",
			serverRes: resp{status: http.StatusRequestEntityTooLarge, body: map[string]string{"error": "large"}},
			wantErr:   false,
		},
		{
			name:      "429 rate limited (async)",
			serverRes: resp{status: http.StatusTooManyRequests, body: map[string]string{"error": "limit"}},
			wantErr:   false,
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
			name:        "deadline exceeded (async, expect no client error)",
			serverRes:   resp{status: http.StatusCreated, body: map[string]string{"message": "ok"}},
			wantErr:     false,
			setTimeout:  50 * time.Millisecond,
			serverDelay: 100 * time.Millisecond,
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

			c := New(srv.URL)
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

			_, err := c.AddEntry(ctx, "user-1", "mem-1", AddEntryRequest{RawEntry: "hello"})
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
