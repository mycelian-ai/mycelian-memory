## Health Checker – Core Spec

### Purpose
- Maintain a cached service health flag set by background component checkers.
- HTTP endpoint is non-blocking and only reads the cached flag.

### Interfaces
```go
type HealthChecker interface {
  Name() string
  IsHealthy() bool                  // cached, non-blocking
  Start(ctx context.Context, interval time.Duration)
}

// Optional fast path implemented by components.
type HealthPinger interface {
  HealthPing(ctx context.Context) error
}
```

### Component Checkers
- StoreHealthChecker
  - Prefer HealthPinger if the store implements it.
  - Else: PingContext on underlying *sql.DB when available.
  - Else: trivial read (e.g., Users().Get("__health_check__")); sql.ErrNoRows counts as healthy.
  - On failure: mark unhealthy and log error with stack; include `checker` name.
- SearchIndexHealthChecker
  - Prefer HealthPinger when the index provides one (e.g., Weaviate GET /v1/meta).
  - Else: issue a cheap no-op (e.g., DeleteVault with empty inputs).
  - On failure: mark unhealthy and log error with stack; include `checker` name.
- EmbedderHealthChecker
  - Prefer HealthPinger if the provider implements it.
  - Else: call Embed("health-check") and require a non-empty vector.
  - On failure: mark unhealthy and log error with stack; include `checker` name.

Notes
- All component checkers use the same polling interval.
- Each probe is bounded by a per-probe timeout.
- Only register checkers for configured dependencies.

### Service Health Checker
```go
type ServiceHealthChecker struct {
  IsHealthy() bool
  Start(ctx context.Context, interval time.Duration)
}
```
- On each tick: if any dependency IsHealthy() is false → service unhealthy; else healthy.
- Logs transitions (UP at info, DOWN at error). No error aggregation.

### HTTP Endpoint Contract
- GET /api/health
  - Always 200 with JSON body: {"status":"healthy"|"unhealthy","timestamp":RFC3339}.
  - Endpoint performs no live checks; it reads the cached flag only.
  - 500 indicates handler failure, not service health.

### Configuration
- Shared polling interval: MEMORY_SERVER_HEALTH_INTERVAL_SECONDS (default 30s).
- Per-probe timeout: MEMORY_SERVER_HEALTH_PROBE_TIMEOUT_SECONDS (default 2s).
- Unconfigured dependencies are skipped.

### Startup behavior
- At startup, component and service health checkers begin running. All checkers start in the unhealthy state until their first successful probe.
- The server waits up to one health interval for the service to become healthy before serving. If still unhealthy after the deadline, it does not start and exits with an error.
