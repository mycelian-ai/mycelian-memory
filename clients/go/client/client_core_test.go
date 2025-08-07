package client

import (
    "context"
    "errors"
    "testing"

    "github.com/mycelian/mycelian-memory/clients/go/client/internal/shardqueue"
)

type stubExec struct{ stops int }

func (s *stubExec) Submit(context.Context, string, shardqueue.Job) error { return nil }
func (s *stubExec) Stop() { s.stops++ }

func TestIsBackPressure(t *testing.T) {
    if !IsBackPressure(ErrBackPressure) {
        t.Fatalf("expected back pressure")
    }
    if IsBackPressure(errors.New("other")) {
        t.Fatalf("unexpected back pressure detection")
    }
}

func TestCloseIdempotent(t *testing.T) {
    s := &stubExec{}
    c := &Client{exec: s}
    if err := c.Close(); err != nil {
        t.Fatalf("close: %v", err)
    }
    if err := c.Close(); err != nil {
        t.Fatalf("second close: %v", err)
    }
    if s.stops != 1 {
        t.Fatalf("executor stop called %d times", s.stops)
    }
}

func TestMustNew(t *testing.T) {
    if MustNew("http://example.com") == nil {
        t.Fatalf("expected client")
    }
}
