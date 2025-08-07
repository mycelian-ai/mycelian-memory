package job

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestJobFunc_NilGuard(t *testing.T) {
	t.Parallel()
	var jf jobFunc // nil
	if err := jf.Run(context.Background()); !errors.Is(err, ErrNilJobFunc) {
		t.Fatalf("expected ErrNilJobFunc, got %v", err)
	}
}

func TestJobFunc_RunSuccessUsingNew(t *testing.T) {
	t.Parallel()
	type ctxKey string
	key := ctxKey("k")
	ctx := context.WithValue(context.Background(), key, "v")

	called := false
	jf := New(func(c context.Context) error {
		called = true
		if got, ok := c.Value(key).(string); !ok || got != "v" {
			return fmt.Errorf("context value mismatch: %v", c.Value(key))
		}
		return nil
	})

	if err := jf.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected wrapped function to be called")
	}
}

func TestJobFunc_RunErrorPropagation(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	jf := New(func(context.Context) error { return sentinel })
	if err := jf.Run(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
