// Copyright 2025 The Synapse Authors.
//
// Package shardqueue provides a lightweight sharded work‑queue that guarantees
// FIFO order *per key* while allowing parallelism across shards.
//
// **Contract**: Callers **must not** invoke Submit concurrently for the *same*
// key.  FIFO ordering relies on that external serialisation.
package shardqueue

import (
	"context"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/mycelian/mycelian-memory/client/internal/errors"
)

type queuedJob struct {
	ctx context.Context
	job Job
}

// ShardExecutor executes Jobs on worker goroutines partitioned by a stable hash
// of the key (e.g. memoryID).  FIFO ordering is preserved within a shard; jobs
// with different keys may run in parallel.
type ShardExecutor struct {
	cfg    Config
	queues []chan queuedJob // len == cfg.Shards

	done   chan struct{} // closed in Stop()
	closed uint32        // 0 → running, 1 → closed

	wg sync.WaitGroup
}

// NewShardExecutor constructs the executor and starts its shard workers.
func NewShardExecutor(cfg Config) *ShardExecutor {
	// Apply zero‑value defaults.
	if cfg.Shards <= 0 {
		cfg.Shards = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 128
	}
	if cfg.EnqueueTimeout <= 0 {
		cfg.EnqueueTimeout = 100 * time.Millisecond
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 8
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 100 * time.Millisecond
	}
	if cfg.MaxInterval <= 0 {
		cfg.MaxInterval = 20 * time.Second
	}

	p := &ShardExecutor{
		cfg:    cfg,
		queues: make([]chan queuedJob, cfg.Shards),
		done:   make(chan struct{}),
	}
	for i := 0; i < cfg.Shards; i++ {
		ch := make(chan queuedJob, cfg.QueueSize)
		p.queues[i] = ch
		p.wg.Add(1)
		go p.runWorker(i, ch)
	}
	return p
}

// Submit enqueues job for the shard derived from key.
//
//   - Returns nil on success.
//   - Returns ErrExecutorClosed if the executor is stopped.
//   - Returns ErrQueueFull (wrapped in *QueueFullError) if the shard is full
//     after EnqueueTimeout elapses.
//   - Returns ctx.Err() if the caller‑provided context is cancelled first.
func (p *ShardExecutor) Submit(ctx context.Context, key string, job Job) error {
	// Fast checks to avoid accepting work after Stop().
	// 1. If Stop() has set the flag but not yet closed p.done we still reject.
	if atomic.LoadUint32(&p.closed) == 1 {
		return ErrExecutorClosed
	}

	// 2. Complementary check: p.done may already be closed even if we missed
	// the flag change.
	select {
	case <-p.done:
		return ErrExecutorClosed
	default:
	}

	qj := queuedJob{ctx: ctx, job: job}
	shard := p.shardFor(key)
	ch := p.queues[shard]

	timer := time.NewTimer(p.cfg.EnqueueTimeout)
	defer timer.Stop()

	select {
	case ch <- qj:
		submissionsTotal.WithLabelValues(labelFor(shard)).Inc()
		return nil

	case <-p.done: // Stop() may be called while waiting for space
		return ErrExecutorClosed

	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		queueFullTotal.WithLabelValues(labelFor(shard)).Inc()
		return &QueueFullError{
			Shard:    shard,
			Length:   len(ch),
			Capacity: cap(ch),
		}
	}
}

// Barrier enqueues a no-op job on the shard for key and waits until it runs,
// ensuring all previously submitted jobs for that key have completed.
func (p *ShardExecutor) Barrier(ctx context.Context, key string) error {
	done := make(chan struct{})
	// Reuse JobFunc adapter to avoid exposing details to callers.
	j := JobFunc(func(context.Context) error {
		close(done)
		return nil
	})
	if err := p.Submit(ctx, key, j); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// Stop signals every worker to finish draining its current queue, waits for
// them to terminate, and then returns.  It is idempotent and safe for
// concurrent use.
func (p *ShardExecutor) Stop() {
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return // already closed
	}

	// Log start of graceful shutdown
	log.Printf("shardqueue: stopping executor, draining %d shards", p.cfg.Shards)

	close(p.done)
	p.wg.Wait()

	// Log completion of graceful shutdown
	log.Printf("shardqueue: executor stopped, all queues drained")
}

// Close lets ShardExecutor satisfy io.Closer.
func (p *ShardExecutor) Close() error {
	p.Stop()
	return nil
}

// ------------------------- internals -------------------------

func (p *ShardExecutor) runWorker(idx int, ch <-chan queuedJob) {
	defer p.wg.Done()

	// Protect worker from crashing the entire executor.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("shardqueue: worker %d panic: %v", idx, r)
		}
	}()

	label := labelFor(idx)

	for {
		select {
		case qj := <-ch:
			if qj.job == nil {
				continue
			}

			// Honour caller context so a cancelled job doesn't stall the shard.
			select {
			case <-qj.ctx.Done():
				p.safeHandleError(qj.ctx.Err())
				// Do not record latency for a job we didn't run.
			default:
				attempts := 0
				exp := backoff.NewExponentialBackOff()
				exp.InitialInterval = p.cfg.BaseBackoff
				exp.Multiplier = 2
				exp.MaxInterval = p.cfg.MaxInterval
				exp.Reset()

				var err error
				for {
					start := time.Now()
					err = qj.job.Run(qj.ctx)
					runDuration.WithLabelValues(label).Observe(time.Since(start).Seconds())

					if err == nil {
						break // Success - exit retry loop
					}

					// Check if this is an irrecoverable error (fail fast)
					if isIrrecoverableError(err) {
						p.safeHandleError(err)
						break // Don't retry irrecoverable errors
					}

					// Retry logic for recoverable errors
					if attempts >= p.cfg.MaxAttempts-1 {
						p.safeHandleError(err) // Max retries exceeded
						break
					}

					attempts++
					wait := exp.NextBackOff()
					select {
					case <-time.After(wait):
					case <-p.done:
						return
					case <-qj.ctx.Done():
						p.safeHandleError(qj.ctx.Err())
						attempts = p.cfg.MaxAttempts // force exit loop
					}
				}
			}

			queueDepth.WithLabelValues(label).Set(float64(len(ch)))

		case <-p.done:
			// Drain remaining jobs, preserving FIFO, then exit.
			remainingJobs := len(ch)
			if remainingJobs > 0 {
				log.Printf("shardqueue: worker %d draining %d remaining jobs", idx, remainingJobs)
			}

			drained := 0
			for {
				select {
				case qj := <-ch:
					if qj.job != nil {
						_ = qj.job.Run(qj.ctx)
						drained++
					}
				default:
					if drained > 0 {
						log.Printf("shardqueue: worker %d drained %d jobs", idx, drained)
					}
					queueDepth.WithLabelValues(label).Set(0)
					return
				}
			}
		}
	}
}

func (p *ShardExecutor) safeHandleError(err error) {
	if err == nil || p.cfg.ErrorHandler == nil {
		return
	}
	func() {
		// Guard against panics in the user‑supplied handler.
		defer func() {
			if r := recover(); r != nil {
				log.Printf("shardqueue: error handler panic: %v", r)
			}
		}()
		p.cfg.ErrorHandler(err)
	}()
}

func (p *ShardExecutor) shardFor(key string) int {
	h := fnv.New32a() // fast and sufficient at our scale
	_, _ = h.Write([]byte(key))
	return int(h.Sum32()) % p.cfg.Shards
}

// isIrrecoverableError checks if an error should not be retried.
func isIrrecoverableError(err error) bool {
	return errors.IsIrrecoverable(err)
}
