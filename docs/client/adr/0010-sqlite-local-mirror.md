---
adr: 0010-sqlite-local-mirror
status: accepted
date: 2025-07-26
---
# SQLite Local Mirror – Crash Consistency, Performance & Cleanup

## Context
Each Synapse MCP instance keeps a local SQLite WAL database that mirrors a subset of user data for sub-millisecond writes and offline support.  The canonical source of truth remains the remote Spanner backend, so the mirror is a **cache with durability expectations** (must not silently lose unsynced rows).

We need clear policies for:
1. Crash-consistency & corruption recovery.
2. Performance tuning on personal devices.
3. Size / age based cleanup so the DB never fills the user's disk.

## Decision
### 1. Crash-Consistency & Corruption Handling
* SQLite runs in **WAL mode** with `synchronous=NORMAL` (durable WAL fsync, fast commits).
* On startup:
  ```go
  PRAGMA journal_mode=WAL;
  PRAGMA quick_check;          -- fast structural check
  if result != "ok" { rename db -> db.broken-<ts>; init fresh DB }
  ```
* Background: daily `PRAGMA integrity_check;` (deep scan).  On error → log, alert metric, auto-rebuild as above.
* Rebuild strategy: discard corrupt file, create empty DB, replicator will re-hydrate unsynced rows from Spanner when accessed.

### 2. Performance Best-Practices
* **SSD / NVMe recommended** – single-row INSERT+COMMIT p50 ≈ 0.2 ms, vs 5-15 ms on HDD.
* Connection pool: `SetMaxOpenConns(64)`; shard jobs borrow a connection per TX.
* `PRAGMA wal_autocheckpoint = 1000;` – keeps WAL segments small.
* `PRAGMA temp_store = MEMORY;` – avoids temp files on disk.

### 3. Size & Age Based Cleanup
Env vars with sensible defaults:
```
SQLITE_RETENTION_DAYS=30      # purge old replicated rows
SQLITE_MAX_SIZE_MB=500        # hard cap disk usage
SQLITE_PURGE_CHUNK=10000      # rows per delete batch
```
Cleanup cron (daily):
1. Purge replicated rows older than `RETENTION_DAYS`.
2. If DB size still > `MAX_SIZE_MB`, delete oldest replicated rows in `PURGE_CHUNK` loops until below cap.
3. After purge:
   ```sql
   PRAGMA wal_checkpoint(PASSIVE);
   PRAGMA incremental_vacuum(1000);
   ```
4. Metrics: `sqlite_purge_rows_total`, `sqlite_db_size_mb`, `sqlite_integrity_errors_total`.

## Consequences
+ Sub-millisecond write ACKs on SSD hardware.
+ Automatic recovery from file corruption; worst-case data loss limited to unsynced rows.
+ Local DB size bounded (age + space) – ≤500 MB even for power users with 1 M entries.
+ Operators can tune retention/size with env vars; sensible defaults need no manual tweaking.

## Alternatives Considered
| Alternative | Pros | Cons |
|-------------|------|------|
| Full WAL `synchronous=FULL` | Max durability | +1 ms latency per write, unnecessary given remote canonical copy |
| Keep all history indefinitely | Simpler | Disk bloat on long-running devices |
| Use LiteFS / dqlite replication | Transparent failover | Overkill for single-user desktop use, adds network & raft complexity |

_Status: Accepted – implementation scheduled alongside shard worker roll-out (Milestone 2B)._ 