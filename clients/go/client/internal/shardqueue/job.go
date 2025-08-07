package shardqueue

import "context"

// Job is a unit of work executed by a ShardExecutor.
// Run must be safe for concurrent invocations when the same Job instance is reused.
type Job interface {
	Run(ctx context.Context) error
}
