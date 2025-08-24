package shardqueue

import (
	"context"
	"testing"
)

func TestJobFunc_AdaptsFunction(t *testing.T) {
	called := false
	j := JobFunc(func(ctx context.Context) error { called = true; return nil })
	if err := j.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !called {
		t.Fatal("expected function to be called")
	}
}
