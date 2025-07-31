---
description: Defer local SQLite mirror; use filesystem snapshots for context & rely on remote API for durability
---
# ADR-0018: Defer SQLite Local Mirror to Post-MVP Roadmap

| Status | Date | Deciders |
| ------ | ---- | -------- |
| ✅ Accepted | 2025-07-27 | @core-team |

## Context

ADR-0010 introduced a local SQLite WAL mirror to queue writes offline and enforce per-memory ordering. Subsequent discussion revealed a simpler MVP path:

1. Many early users will operate online; sub-5 ms ACKs are not yet critical.
2. The filesystem itself offers strong local consistency for context files.
3. A single (or 4-bucket) in-memory executor queue already guarantees FIFO ordering.

Maintaining SQLite brings notable complexity—schema migration, WAL tuning, disk cleanup, PII at rest, corruption recovery—before we have evidence it is required.

## Decision

* Remove the SQLite mirror from the steel-thread implementation (Milestone 2).
* All state-changing operations will be sent directly to the HTTP backend.
* Context management protocol:
  * Context files live under `~/.synapse/contexts/<memoryId>/`.
  * Each write event (AddEntry, DeleteEntry, etc.) includes a zip snapshot of that directory (`contextZip`).
  * FIFO executor guarantees the snapshot is consistent with the entry.
* The executor remains hash-partitioned (default 4 buckets) with buffered channels for back-pressure.

## Consequences

Positive
* Codebase slimmer: ~1 kLOC of DB handling, migrations, and purge logic eliminated.
* Operational posture simpler: no local DB corruption, no size limits, easier compliance.
* Quick iteration: we can ship Milestone 2 without dealing with disk I/O edge cases.

Negative / Risks
* Online dependency: writes now wait on HTTP latency; bursts may be slower until we add back pressure tuning.
* No offline write capability until the mirror is re-introduced.

## Migration Plan (Future Roadmap)

1. Collect real telemetry (queue depth, latency) to decide when offline queue is necessary.
2. If/when required, re-enable SQLite behind the existing `Job` executor interface.
3. Gate the feature via `--with-local-mirror` flag and gradual rollout.

## Alternatives Considered
| Option | Reason Rejected |
| ------ | --------------- |
| Keep SQLite from day one | Complexity outweighs current benefits; MVP latency acceptable with direct HTTP. |
| Use embedded BoltDB | Similar operational burden, weaker ecosystem tools. |

## References
* ADR-0010 SQLite Local Mirror (superseded for MVP) 