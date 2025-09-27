# Outbox Worker Reliability Review

## Overview
This note captures reliability risks observed in the outbox worker implementation (`cmd/outbox-worker`, `server/outboxworker`, and `server/internal/outbox`) and outlines concrete remediation steps. The goal is to make the worker resilient to dependency failures (embeddings service, Weaviate), keep Postgres healthy, and ensure issues surface rapidly to operators.

## Recent Updates
- Added configurable timeouts for startup embedding checks (default 10 s), per-job embeds (12 s), and index mutations (5 s) so dependency hangs surface quickly.
- Surface Weaviate bootstrap failures and close Postgres connections cleanly to avoid hidden startup issues.

## Identified Reliability Risks
- **Unbounded startup probe against the embedder** (`server/outboxworker/run.go:48`)
  - The worker runs `emb.Embed(context.Background(), "worker-startup-check")` without a timeout. If the embed provider hangs or the network path is wedged, startup blocks forever and supervisors cannot restart cleanly.
- **Silent Weaviate bootstrap failures** (`server/outboxworker/run.go:53`)
  - Errors from `searchindex.BootstrapWeaviate` are ignored. When schema creation fails (Weaviate offline/incompatible), the worker keeps running, every job fails later, and the queue backs off without a clear root cause.
- **External calls share the long-lived worker context** (`server/internal/outbox/worker.go:185`, `:209`, `:197`, `:214`)
  - Each embed/index/delete call uses the worker’s top-level context, which has no deadline. The Weaviate client defaults to an `http.Client` without timeouts, so a single stalled HTTP request blocks the whole batch and holds database resources during retry/backoff windows.
- **Database transaction spans network work** (`server/internal/outbox/worker.go:102-137`)
  - The worker starts a SQL transaction, leases jobs, and then performs embedding/indexing inside the same transaction. Slow dependencies hold Postgres row locks and the connection for the entire batch, amplifying contention and making multi-worker scaling risky.

## Recommendations
1. **Bound critical dependency checks**
   - Wrap the startup embed probe in `context.WithTimeout` (e.g., 5–10s) and return an error when the deadline elapses so the process exits and can be restarted.
2. **Fail fast on bootstrap errors**
   - Check the `BootstrapWeaviate` return value and handle it (log + exit). Optionally add retry/backoff before bailing to handle transient outages explicitly.
3. **Introduce per-operation deadlines**
   - Derive short-lived contexts for embeds and index mutations. Suggested defaults:
     - 10 s for the startup embed probe to mirror the Ollama client timeout (`server/internal/embeddings/ollama/ollama.go`).
    - 12 s for per-job embeds so hung inference calls fail and requeue before the retry backoff hits its 1-minute cap.
     - 5 s for search-index writes/deletes; they normally complete in <500 ms, so the timeout catches network stalls while keeping retries fast.
   - Treat deadline or cancellation as retryable errors so jobs requeue quickly without blocking the batch. The existing logging already records these errors as `context deadline exceeded`, but we can add structured fields (e.g., `reason=timeout`) if we want clearer dashboards.
4. **Decouple DB transactions from external work**
    - Current implementation still performs embedding/indexing inside the transaction and relies on row locks to provide at-least-once semantics. If dependency latency becomes an issue, we should either mark rows as `processing` with leases or commit between jobs to shorten the lock window.
5. **Strengthen verification and observability**
   - Extend `server/internal/outbox` tests beyond the `handle` helper: add coverage for `processOnce`, lease/backoff behaviour, and delete operations (including failure paths) so the state machine is guarded by automated checks.
   - Continue enriching logs (attempt counters, retry schedules, elapsed duration) to speed incident triage; keep debug-only payload logging behind config so sensitive text stays out of production logs.
6. **Raise code quality and maintainability**
   - Refactor the worker loop so database work and external calls are clearly separated (e.g., fetch jobs, commit, then process with per-job contexts). That eliminates long-lived transactions and makes the control flow easier to reason about.
   - Extract the retry/backoff mechanics into a small helper type (e.g., `LeaseManager`) with dedicated tests, reducing duplication between SQL and Go and making future policy tweaks safer.
   - Document configuration knobs (timeouts, batch size, backoff cap) in `server/internal/config` so defaults and tuning guidance live alongside the code.

Implementing these changes will shorten failure detection loops, improve throughput under dependency degradation, and keep the outbox pipeline aligned with the reliability-first priority in `CLAUDE.md`.
