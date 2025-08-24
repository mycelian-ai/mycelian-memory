package shardqueue

import (
	"context"
	"testing"
	"time"
)

// A panic in one shard worker should not crash other shards; jobs on other shards continue to run.
func TestWorker_PanicDoesNotStopOtherShards(t *testing.T) {
	ex := NewShardExecutor(Config{Shards: 2, QueueSize: 4, MaxAttempts: 1})
	defer ex.Stop()

	// Choose two keys that map to different shards.
	keyPanic := "panic-key"
	shardPanic := ex.shardFor(keyPanic)
	keyOther := "other-key"
	for tries := 0; tries < 100 && ex.shardFor(keyOther) == shardPanic; tries++ {
		keyOther = keyOther + "x"
	}
	if ex.shardFor(keyOther) == shardPanic {
		t.Fatal("failed to find keys mapping to different shards")
	}

	// Submit a job that panics on shardPanic.
	if err := ex.Submit(context.Background(), keyPanic, JobFunc(func(ctx context.Context) error { panic("job panic") })); err != nil {
		t.Fatalf("submit panic job: %v", err)
	}

	// Submit a job on the other shard; it should still run even if the panic kills one worker.
	ran := make(chan struct{})
	if err := ex.Submit(context.Background(), keyOther, JobFunc(func(ctx context.Context) error { close(ran); return nil })); err != nil {
		t.Fatalf("submit follow-up: %v", err)
	}

	select {
	case <-ran:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("other shard did not continue after worker panic")
	}
}
