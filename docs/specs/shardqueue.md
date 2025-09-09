# ShardQueue Implementation Specification

**Package**: `client/internal/shardqueue`
**Version**: 1.0
**Date**: January 2025
**Status**: Stable

## Overview

The `shardqueue` package provides a lightweight sharded work queue that guarantees **FIFO order per key** while allowing parallelism across different keys. It serves as the core execution engine for ordered operations in the Mycelian Memory client SDK.

## Design Goals

1. **Per-key FIFO ordering**: All jobs for the same key execute sequentially
2. **Cross-key parallelism**: Different keys can execute concurrently across shards
3. **Bounded resource usage**: Configurable worker count and queue sizes
4. **Resilience**: Automatic retry with exponential backoff
5. **Graceful degradation**: Back-pressure handling and clean shutdown
6. **Observability**: Comprehensive Prometheus metrics

## Core Architecture

### ShardExecutor

The main component that manages the sharded work queue:

```go
type ShardExecutor struct {
    cfg    Config
    queues []chan queuedJob  // len == cfg.Shards
    done   chan struct{}     // closed in Stop()
    closed uint32           // atomic flag: 0 → running, 1 → closed
    wg     sync.WaitGroup
}
```

### Job Interface

Work units must implement the `Job` interface:

```go
type Job interface {
    Run(ctx context.Context) error
}

// Helper adapter for function closures
type JobFunc func(ctx context.Context) error
func (f JobFunc) Run(ctx context.Context) error { return f(ctx) }
```

### Configuration

Environment-driven configuration with `SQ_` prefix:

```go
type Config struct {
    Shards         int           `envconfig:"SHARDS"          default:"4"`
    QueueSize      int           `envconfig:"QUEUE_SIZE"      default:"128"`
    EnqueueTimeout time.Duration `envconfig:"ENQUEUE_TIMEOUT" default:"100ms"`
    MaxAttempts    int           `envconfig:"MAX_ATTEMPTS"    default:"8"`
    BaseBackoff    time.Duration `envconfig:"BASE_BACKOFF"    default:"100ms"`
    MaxInterval    time.Duration `envconfig:"MAX_INTERVAL"    default:"20s"`
    ErrorHandler   func(error)   `envconfig:"-"`
}
```

**Environment Variables**:
- `SQ_SHARDS=8` - Number of worker goroutines
- `SQ_QUEUE_SIZE=256` - Buffer size per shard queue
- `SQ_ENQUEUE_TIMEOUT=200ms` - Timeout for queue full conditions
- `SQ_MAX_ATTEMPTS=5` - Maximum retry attempts per job
- `SQ_BASE_BACKOFF=50ms` - Initial retry delay
- `SQ_MAX_INTERVAL=30s` - Maximum retry delay

## API Specification

### Constructor

```go
func NewShardExecutor(cfg Config) *ShardExecutor
```

**Behavior**:
- Applies default values for zero-valued config fields
- Creates `cfg.Shards` worker goroutines immediately
- Each worker has a dedicated buffered channel of size `cfg.QueueSize`
- Workers start running immediately and process jobs until `Stop()` is called

### Job Submission

```go
func (p *ShardExecutor) Submit(ctx context.Context, key string, job Job) error
```

**Parameters**:
- `ctx`: Caller context (cancellation propagated to job execution)
- `key`: Routing key (e.g., memoryID) for shard selection
- `job`: Work unit to execute

**Returns**:
- `nil`: Job successfully enqueued
- `ErrExecutorClosed`: Executor has been stopped
- `*QueueFullError`: Shard queue full after `EnqueueTimeout`
- `ctx.Err()`: Caller context cancelled while waiting

**Shard Selection**: Uses FNV32a hash for deterministic, stable routing:
```go
func (p *ShardExecutor) shardFor(key string) int {
    h := fnv.New32a()
    h.Write([]byte(key))
    return int(h.Sum32()) % p.cfg.Shards
}
```

**Critical Contract**: Callers **must not** invoke `Submit` concurrently for the same key. FIFO ordering relies on external serialization.

### Lifecycle Management

```go
func (p *ShardExecutor) Stop()
func (p *ShardExecutor) Close() error  // implements io.Closer
```

**Stop Behavior**:
- Idempotent and safe for concurrent calls
- Sets atomic `closed` flag and closes `done` channel
- Workers finish current job, then drain remaining queued jobs
- Preserves FIFO order during shutdown
- Blocks until all workers terminate
- Logs shutdown progress

## Worker Execution Model

Each worker goroutine runs this loop:

1. **Job Dequeue**: Blocks on channel receive or shutdown signal
2. **Context Check**: Skip execution if caller context already cancelled
3. **Retry Loop**: Execute job with exponential backoff on failure
4. **Metrics Update**: Record execution time and queue depth
5. **Error Handling**: Call configured error handler (if provided)

### Retry Mechanism

```go
// Exponential backoff configuration
exp := backoff.NewExponentialBackOff()
exp.InitialInterval = p.cfg.BaseBackoff  // default: 100ms
exp.Multiplier = 2
exp.MaxInterval = p.cfg.MaxInterval      // default: 20s
```

**Retry Logic**:
- Retry up to `MaxAttempts` times (default: 8)
- Double delay between attempts (capped at `MaxInterval`)
- Stop retrying if job context is cancelled
- Stop retrying if executor is shutting down
- Call `ErrorHandler` for final failure (if configured)

### Panic Recovery

Workers are protected by panic recovery:
```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("shardqueue: worker %d panic: %v", idx, r)
    }
}()
```

## Error Types

### ErrExecutorClosed
```go
var ErrExecutorClosed = errors.New("shard executor closed")
```
Permanent condition: executor stopped, no further work accepted.

### ErrQueueFull
```go
var ErrQueueFull = errors.New("shard queue full")

type QueueFullError struct {
    Shard    int // 0 ≤ Shard < cfg.Shards
    Length   int // queue length at timeout
    Capacity int // cap(queue)
}
```
Transient back-pressure: shard queue full when `Submit` attempted enqueue.

## Observability

### Prometheus Metrics

All metrics use namespace `mycelian` and subsystem `shardqueue`:

```go
// Job submissions successfully enqueued
mycelian_shardqueue_submissions_total{shard="0"}

// Enqueue timeouts due to full queues
mycelian_shardqueue_queue_full_total{shard="0"}

// Job execution duration histogram
mycelian_shardqueue_run_duration_seconds{shard="0"}

// Current queue depth per shard
mycelian_shardqueue_queue_depth{shard="0"}
```

### Logging

- Shutdown progress: `"shardqueue: stopping executor, draining N shards"`
- Worker panics: `"shardqueue: worker N panic: <error>"`
- Queue draining: `"shardqueue: worker N draining N remaining jobs"`
- Error handler panics: `"shardqueue: error handler panic: <error>"`

## Usage Patterns

### Basic Usage

```go
// Create executor with defaults
cfg := shardqueue.Config{}
exec := shardqueue.NewShardExecutor(cfg)
defer exec.Stop()

// Submit work
job := shardqueue.JobFunc(func(ctx context.Context) error {
    // Do work here
    return doSomeWork(ctx)
})

err := exec.Submit(context.Background(), "memory-123", job)
if err != nil {
    // Handle submission error
}
```

### With Environment Configuration

```go
// Load from environment (SQ_* variables)
cfg, err := shardqueue.LoadConfig()
if err != nil {
    log.Fatal(err)
}

// Override specific settings
cfg.ErrorHandler = func(err error) {
    log.Printf("job failed: %v", err)
}

exec := shardqueue.NewShardExecutor(cfg)
```

### Back-Pressure Handling

```go
err := exec.Submit(ctx, key, job)
switch {
case errors.Is(err, shardqueue.ErrQueueFull):
    var qfe *shardqueue.QueueFullError
    if errors.As(err, &qfe) {
        log.Printf("Shard %d full: %d/%d", qfe.Shard, qfe.Length, qfe.Capacity)
    }
    // Handle back-pressure (retry, drop, 429 response, etc.)

case errors.Is(err, shardqueue.ErrExecutorClosed):
    // Executor shutting down, stop submitting work

case err != nil:
    // Other error (likely context cancellation)
}
```

## Performance Characteristics

### Throughput
- **Single worker**: Sustained >1,000 jobs/second on laptop hardware
- **Default 4 workers**: Linear scaling across different keys
- **Bottleneck**: Individual job execution time, not queue overhead

### Latency
- **Enqueue latency**: Microseconds (channel send)
- **Execution latency**: Job-dependent + network I/O for remote calls
- **Queue full timeout**: 100ms default (configurable)

### Memory Usage
- **Per shard**: `QueueSize * sizeof(queuedJob)` ≈ `128 * 32` bytes = 4KB
- **Default 4 shards**: ~16KB queue buffers + goroutine overhead

### Concurrency Limits
- **Workers**: Configurable via `SQ_SHARDS` (default: 4)
- **Queue depth**: Configurable via `SQ_QUEUE_SIZE` (default: 128)
- **No global limits**: Parallelism scales with number of unique keys

## Integration with Client SDK

The shardqueue serves as the execution engine for the **Ordered (SQ)** concurrency class:

### Ordered Operations (via ShardQueue)
- `add_entry` - Add memory entry
- `delete_entry` - Delete memory entry
- `put_context` - Update memory context
- `await_consistency` - Wait for pending operations

### Key Benefits
- **Read-your-write consistency**: Per-memory FIFO guarantees
- **Offline resilience**: Queued operations retry when connectivity returns
- **Bounded resources**: No unlimited goroutine creation
- **Deterministic debugging**: Predictable operation ordering

## Thread Safety

- **Submit**: Safe for concurrent calls with **different keys**
- **Submit**: **NOT safe** for concurrent calls with the **same key**
- **Stop/Close**: Safe for concurrent calls (idempotent)
- **Config**: Read-only after construction (safe)
- **Metrics**: Thread-safe (Prometheus client handles synchronization)

## Best Practices

### Key Selection
- Use stable, deterministic keys (e.g., UUIDs, database IDs)
- Avoid keys with poor hash distribution
- Consider key cardinality vs shard count for load balancing

### Error Handling
- Always handle `ErrQueueFull` - implement back-pressure strategy
- Set `ErrorHandler` for non-transient error logging/metrics
- Use context cancellation for timeouts and cleanup

### Performance Tuning
- Increase `SQ_SHARDS` for high-throughput scenarios
- Increase `SQ_QUEUE_SIZE` if jobs are bursty
- Monitor `queue_full_total` metric for back-pressure issues
- Monitor `queue_depth` gauge for steady-state utilization

### Testing
- Use short timeouts in tests to avoid long waits
- Test context cancellation behavior
- Test graceful shutdown with pending jobs
- Use stress tests to verify no job overlap per key

## Limitations

1. **No persistence**: Jobs lost on process restart
2. **No job priorities**: All jobs treated equally within a shard
3. **No job reordering**: Strict FIFO per key (by design)
4. **No cross-shard transactions**: Each shard operates independently
5. **No job introspection**: Cannot query job status or remaining work
6. **Memory-bound**: Queue capacity limited by available RAM

## Migration Considerations

Future enhancements may include:
- Persistent queue backends (SQLite, disk files)
- Priority job scheduling within shards
- Cross-shard coordination for transactions
- Job status API for monitoring
- Pluggable retry policies
- Batch job submission APIs

Any such changes must preserve the core FIFO-per-key guarantee that existing code depends on.
