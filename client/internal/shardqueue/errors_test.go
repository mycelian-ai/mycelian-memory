package shardqueue

import (
	"errors"
	"testing"
)

func TestQueueFullError_ErrorAndIs(t *testing.T) {
	e := &QueueFullError{Shard: 3, Length: 10, Capacity: 16}
	if e.Error() == "" {
		t.Fatal("empty error string")
	}
	if !errors.Is(e, ErrQueueFull) {
		t.Fatal("expected errors.Is(e, ErrQueueFull) to be true")
	}
	if errors.Is(e, ErrExecutorClosed) {
		t.Fatal("unexpected match with ErrExecutorClosed")
	}
}
