package client

import (
	"context"
	"errors"
	"testing"
)

func TestJobFunc_NilGuard(t *testing.T) {
	t.Parallel()
	var jf jobFunc // nil
	if err := jf.Run(context.Background()); !errors.Is(err, ErrNilJobFunc) {
		t.Fatalf("expected ErrNilJobFunc, got %v", err)
	}
}
