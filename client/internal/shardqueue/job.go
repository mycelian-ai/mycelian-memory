package shardqueue

import "context"

// Job is a unit of work executed by a ShardExecutor.
// Run must be safe for concurrent invocations when the same Job instance is reused.
type Job interface {
	Run(ctx context.Context) error
}

// JobFunc is a helper to adapt a function to a Job.
type JobFunc func(ctx context.Context) error

// Run implements Job for JobFunc.
func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }
