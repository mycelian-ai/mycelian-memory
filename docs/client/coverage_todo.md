# Client module coverage TODO (goal: >90%)

Work one file at a time. Check items as they land with green tests.

Source of truth for coverage snapshot
- We regenerate from the coverage report produced by:
  - go test ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic
  - go tool cover -func=coverage.out
- Numbers will change as we add tests. This file is updated incrementally from that report.

Current per-file coverage (ascending; from latest coverage.out)
- 0.00%  — `client/internal/shardqueue/config.go`
- 0.00%  — `client/internal/api/prompts.go`
- 7.14%  — `client/internal/shardqueue/errors.go`
- 7.79%  — `client/prompts/embed.go`
- 14.29% — `client/internal/shardqueue/job.go`
- 14.29% — `client/internal/job/sharding.go`
- 14.29% — `client/validate.go`
- 14.29% — `client/client.go`
- 16.19% — `client/options.go`
- 16.90% — `client/internal/api/entries.go`
- 17.71% — `client/internal/api/vaults.go`
- 19.16% — `client/internal/api/contexts.go`
- 19.18% — `client/internal/api/memories.go`
- 20.41% — `client/internal/api/search.go`
- 20.69% — `client/internal/api/users.go`
- 27.99% — `client/internal/shardqueue/shardexecutor.go`
- 39.29% — `client/internal/job/jobfunc.go`
- 42.86% — `client/internal/shardqueue/metrics.go`
- 42.86% — `client/internal/types/validation.go`

- [ ] Target 1: `client/client.go`
  - [ ] Cover `debugTransport.RoundTrip` when DEBUG/MYCELIAN_DEBUG is set (request+response branches)
  - [ ] Cover error path in `RoundTrip` when underlying transport returns error
  - [ ] Cover `newDefaultExecutor` via constructor path (sanity assertion)

- [ ] Target 2: `client/internal/api/users.go`
  - [ ] Add ctx-cancelled case for `CreateUser` (ctx timeout before Do)
  - [ ] Add decode-error case (malformed JSON body)

- [ ] Target 3: `client/internal/api/vaults.go`
  - [ ] Non-OK statuses for `CreateVault`/`ListVaults`/`GetVault`/`DeleteVault`
  - [ ] Input validation failures (bad userID)
  - [ ] `GetVaultByTitle` non-OK status

- [ ] Target 4: `client/internal/api/memories.go`
  - [ ] Non-OK statuses for `CreateMemory`/`ListMemories`/`GetMemory`/`DeleteMemory`
  - [ ] Input validation failures
  - [ ] Decode-error case

- [ ] Target 5: `client/internal/api/entries.go`
  - [ ] Add non-201/204 status in async job to verify error propagation
  - [ ] Input validation failures

- [ ] Target 6: `client/internal/api/contexts.go`
  - [ ] `PutContext` non-201 status error
  - [ ] `GetContext` non-OK non-404 status error

- [ ] Target 7: `client/internal/api/search.go`
  - [ ] Non-OK status error path
  - [ ] Decode-error case

- [ ] Target 8: `client/prompts/embed.go`
  - [ ] `ListMemoryTypes` returns expected set
  - [ ] `LoadDefaultPrompts` unknown type error branch

- [ ] Target 9: `client/options.go`
  - [ ] Auto-enable debug via env in `New` (set `MYCELIAN_DEBUG=true`)
  - [ ] WithDebugLogging(false) leaves transport unchanged

- [ ] Target 10: `client/internal/shardqueue/shardexecutor.go`
  - [ ] ErrorHandler invoked on job error
  - [ ] Drain-on-Stop path exercised (leftover jobs run)

- [ ] Target 11: `client/internal/job/` (jobfunc.go, sharding.go)
  - [ ] `jobfunc.New(nil)` then `Run` returns wrapped `ErrNilJobFunc` and `errors.Is(err, ErrNilJobFunc)` is true
  - [ ] `jobfunc.New` with a real function: function is invoked, receives ctx, and its return value is propagated
  - [ ] `ShardLabel` determinism: same input returns the same label across invocations
  - [ ] `ShardLabel` range: label parses to an integer within [0, 31]
  - [ ] `ShardLabel` distribution sanity: over a sample set (e.g., 1k IDs) at least 20 distinct labels produced

Next up (in order)
- Target 2: `client/internal/api/prompts.go` (0%).
- Target 1: `client/client.go` — specifically `LoadDefaultPrompts` wrapper and `debugTransport` error branch.

Open questions (do not block work)
- Should `client/internal/api/GetVaultByTitle` path be title-lookup or ID? Current path looks like an ID URL; confirm intent later.

Tracking
- Baseline after initial additions: ~65% total. Aim for >90% by completing Targets 1–8; shardqueue extras (Target 10) are stretch to push the remainder.



Command: Top 5 lowest-coverage files (client module)
- Run from repo root or client module; this recomputes coverage and prints the 5 lowest files.

```sh
cd client && \
go test ./... -coverpkg=./... -coverprofile=coverage.out -covermode=atomic && \
awk 'NR>1 {split($1,a,":"); file=a[1]; stmts=$2; count=$3; total[file]+=stmts; if(count>0) covered[file]+=stmts} END {for (f in total) {pct=(covered[f]/total[f])*100; printf("%6.2f%%\t%6d/%-6d\t%s\n", pct, covered[f], total[f], f)}}' coverage.out | \
sort -n -k1,1 | head -5
```

Decision log (skip list)
- Use this to record files we intentionally skip improving (with rationale). Do not remove entries; append with date.
- Template:
  - YYYY-MM-DD — SKIP `path/to/file.go`: reason (e.g., trivial getters; third-party glue; deprecated)

Current evaluation of top-5 (from latest run)
- `client/internal/shardqueue/config.go` — Proceed. Add `LoadConfig` test that sets env vars and verifies defaults.
- `client/internal/api/prompts.go` — Proceed. Test wrapper (ctx-cancel and success path).
- `client/internal/shardqueue/errors.go` — Proceed. Test `QueueFullError.Error` and `errors.Is` behavior.
- `client/prompts/embed.go` — Proceed. Add `ListMemoryTypes` test (we already cover unknown type and load success).
- `client/internal/shardqueue/job.go` — Proceed. Add `JobFunc` adapter test to ensure it calls the function.
