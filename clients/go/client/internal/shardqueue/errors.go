package shardqueue

import (
	"errors"
	"fmt"
)

// ErrQueueFull reports transient back‑pressure: the shard queue was full
// when Submit tried to enqueue a job.
var ErrQueueFull = errors.New("shard queue full")

// ErrExecutorClosed reports a permanent condition: the executor has been
// stopped and will accept no further work.
var ErrExecutorClosed = errors.New("shard executor closed")

// QueueFullError carries diagnostics while satisfying errors.Is(_, ErrQueueFull).
type QueueFullError struct {
	Shard    int // 0 ≤ Shard < cfg.Shards
	Length   int // queue length at timeout
	Capacity int // cap(queue)
}

func (e *QueueFullError) Error() string {
	return fmt.Sprintf("shard queue %d full (len=%d cap=%d)", e.Shard, e.Length, e.Capacity)
}

func (e *QueueFullError) Is(target error) bool { return target == ErrQueueFull }
