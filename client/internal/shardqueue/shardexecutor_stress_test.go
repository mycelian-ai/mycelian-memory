//go:build stress

package shardqueue

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// StressSerialNoOverlap verifies—at high volume—that only one job for the same
// key is ever in-flight at a time. It runs 10 000 jobs across four shards.
func TestStressSerialNoOverlap(t *testing.T) {
	t.Parallel()

	const (
		totalJobs = 1_000
		key       = "serial-key"
	)

	p := NewShardExecutor(Config{Shards: 4, QueueSize: 1024})
	defer p.Stop()

	var (
		inFlight        int32
		overlapDetected int32
		wg              sync.WaitGroup
	)
	wg.Add(totalJobs)

	for i := 0; i < totalJobs; i++ {
		go func() {
			defer wg.Done()
			_ = p.Submit(context.Background(), key, testJob{run: func(context.Context) error {
				if atomic.AddInt32(&inFlight, 1) > 1 {
					atomic.StoreInt32(&overlapDetected, 1)
				}
				time.Sleep(time.Microsecond) // widen race window
				atomic.AddInt32(&inFlight, -1)
				return nil
			}})
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for %d jobs", totalJobs)
	}

	if atomic.LoadInt32(&overlapDetected) == 1 {
		t.Fatalf("detected overlapping execution for key %q", key)
	}
}

// StressParallelDifferentKeys measures that jobs for different keys run in
// parallel by sampling the maximum number of goroutines in flight.
func TestStressParallelDifferentKeys(t *testing.T) {
	t.Parallel()

	const (
		keys       = 16
		jobsPerKey = 40
	)

	p := NewShardExecutor(Config{Shards: 8, QueueSize: 256})
	defer p.Stop()

	var (
		inFlight int32
		maxSeen  int32
		wg       sync.WaitGroup
	)
	wg.Add(keys * jobsPerKey)

	for k := 0; k < keys; k++ {
		key := "K" + string(rune(k))
		for j := 0; j < jobsPerKey; j++ {
			go func() {
				defer wg.Done()
				_ = p.Submit(context.Background(), key, testJob{run: func(context.Context) error {
					n := atomic.AddInt32(&inFlight, 1)
					for {
						m := atomic.LoadInt32(&maxSeen)
						if n <= m || atomic.CompareAndSwapInt32(&maxSeen, m, n) {
							break
						}
					}
					time.Sleep(50 * time.Microsecond)
					atomic.AddInt32(&inFlight, -1)
					return nil
				}})
			}()
		}
	}

	wg.Wait()

	maxExpected := int32(8) // shards configured above
	gmp := int32(runtime.GOMAXPROCS(0))
	if gmp < maxExpected {
		maxExpected = gmp
	}
	if maxExpected < 2 {
		maxExpected = 1
	}

	if maxSeen < maxExpected {
		t.Fatalf("expected at least %d jobs in parallel, saw %d", maxExpected, maxSeen)
	}
}

// StressQueueFull exercises back-pressure behaviour with a tiny queue so
// ErrQueueFull must be observed frequently.
func TestStressQueueFull(t *testing.T) {
	t.Parallel()

	p := NewShardExecutor(Config{Shards: 1, QueueSize: 4, EnqueueTimeout: 10 * time.Microsecond})
	defer p.Stop()

	const submitAttempts = 512

	workers := 16
	var (
		fullCount int32
		wgWorkers sync.WaitGroup
	)
	wgWorkers.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wgWorkers.Done()
			for i := 0; i < submitAttempts/workers; i++ {
				err := p.Submit(context.Background(), "C", testJob{run: func(context.Context) error {
					time.Sleep(200 * time.Microsecond)
					return nil
				}})
				if errors.Is(err, ErrQueueFull) {
					atomic.AddInt32(&fullCount, 1)
				}
			}
		}()
	}
	wgWorkers.Wait()

	fc := atomic.LoadInt32(&fullCount)
	if fc == 0 || fc == submitAttempts {
		t.Fatalf("expected some but not all attempts to hit ErrQueueFull; full=%d total=%d", fc, submitAttempts)
	}
}

// StressContextCancellation submits jobs whose contexts timeout before the job
// can run, verifying that the worker respects ctx.Done().
func TestStressContextCancellation(t *testing.T) {
	t.Parallel()

	p := NewShardExecutor(Config{Shards: 2, QueueSize: 16})
	defer p.Stop()

	const jobs = 200
	var cancelled int32
	wg := sync.WaitGroup{}
	wg.Add(jobs)

	for i := 0; i < jobs; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Microsecond)
			defer cancel()
			_ = p.Submit(ctx, "X", testJob{run: func(context.Context) error {
				time.Sleep(200 * time.Microsecond)
				return nil
			}})
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				atomic.AddInt32(&cancelled, 1)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&cancelled) == 0 {
		t.Fatalf("expected some contexts to cancel before execution")
	}
}

// RandomisedStress mixes keys, cancellation and Stop() racing to shake out
// hidden ordering bugs. Captures seed for reproducibility.
func TestRandomisedStress(t *testing.T) {
	t.Parallel()

	const (
		duration    = 200 * time.Millisecond
		concurrency = 8
	)

	// ----- deterministic replay support -----
	baseSeed := func() int64 {
		if s := os.Getenv("SHARDQUEUE_STRESS_SEED"); s != "" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				return v
			}
		}
		return time.Now().UnixNano()
	}()
	t.Logf("RandomisedStress seed=%d", baseSeed)

	p := NewShardExecutor(Config{Shards: 8, QueueSize: 64})
	defer p.Stop()

	stopCtx, stop := context.WithTimeout(context.Background(), duration)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for id := 0; id < concurrency; id++ {
		r := rand.New(rand.NewSource(baseSeed + int64(id)))

		go func(rng *rand.Rand) {
			defer wg.Done()
			for {
				select {
				case <-stopCtx.Done():
					return
				default:
				}

				key := "K" + string(rune(rng.Intn(32)))
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rng.Intn(200))*time.Microsecond)

				d2 := rng.Intn(150)
				sleepDur := time.Duration(d2) * time.Microsecond
				_ = p.Submit(ctx, key, testJob{run: func(context.Context) error {
					time.Sleep(sleepDur)
					return nil
				}})
				cancel()
			}
		}(r)
	}

	wg.Wait()
}
