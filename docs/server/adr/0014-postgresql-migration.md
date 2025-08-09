---
title: Migrate storage from Spanner/SQLite to PostgreSQL (local) and Aurora PostgreSQL Serverless v2 (prod)
status: Proposed
date: 2025-08-08
---

## Goal
Replace current Spanner (prod) and SQLite (local) with:
- Local/dev/CI: PostgreSQL 16 (docker) for parity with prod
- Production: AWS Aurora PostgreSQL Serverless v2

Drivers for change: single SQL dialect, easier local setup, full feature parity (JSONB, SKIP LOCKED), and readiness for outbox → Weaviate.

## Scope (what changes)
- New Postgres storage adapter (pgx) implementing `storage.Storage`
- Config to select `DB_DRIVER=postgres` and DSN env (`DATABASE_URL`)
- SQL schema (DDL) in Postgres dialect; migrations for create/update
- Transactional Outbox table + worker using `FOR UPDATE SKIP LOCKED`
- Docker Compose service for postgres in local stack; retire Spanner emulator + SQLite in CI
- Update server tests to run against Postgres

## Data Model (PostgreSQL DDL)
Notes: Use TIMESTAMPTZ for times, JSONB for documents, UUID type or TEXT (client generates UUIDs; either works). Below uses TEXT for minimal friction; can switch to UUID later.

```sql
-- Users
CREATE TABLE IF NOT EXISTS users (
  user_id        TEXT PRIMARY KEY,
  email          TEXT NOT NULL UNIQUE,
  display_name   TEXT,
  time_zone      TEXT NOT NULL DEFAULT 'UTC',
  status         TEXT NOT NULL DEFAULT 'ACTIVE',
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_active_time TIMESTAMPTZ
);

-- Vaults
CREATE TABLE IF NOT EXISTS vaults (
  user_id        TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  title          TEXT NOT NULL,
  description    TEXT,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, vault_id),
  UNIQUE (user_id, title)
);

-- Memories
CREATE TABLE IF NOT EXISTS memories (
  user_id        TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  memory_type    TEXT NOT NULL,
  title          TEXT NOT NULL,
  description    TEXT,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, vault_id, memory_id),
  UNIQUE (vault_id, title)
);

-- MemoryEntries: chronological PK; unique entry_id for fast lookup
-- Hard deletes only (no expiration_time/TTL)
CREATE TABLE IF NOT EXISTS memory_entries (
  user_id        TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  title          TEXT,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  entry_id       TEXT NOT NULL,
  raw_entry      TEXT NOT NULL,
  summary        TEXT,
  metadata       JSONB,
  tags           JSONB,
  correction_time TIMESTAMPTZ,
  corrected_entry_memory_id TEXT,
  corrected_entry_creation_time TIMESTAMPTZ,
  correction_reason TEXT,
  last_update_time TIMESTAMPTZ,
  PRIMARY KEY (user_id, vault_id, memory_id, creation_time, entry_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS memory_entries_entry_id_uq ON memory_entries(entry_id);
CREATE INDEX IF NOT EXISTS memory_entries_recent_idx ON memory_entries(user_id, vault_id, memory_id, creation_time DESC);

-- MemoryContexts: append-only snapshots
CREATE TABLE IF NOT EXISTS memory_contexts (
  user_id        TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  context_id     TEXT NOT NULL,
  context        JSONB NOT NULL,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, vault_id, memory_id, context_id)
);

-- Outbox for Weaviate sync (idempotent worker)
CREATE TABLE IF NOT EXISTS outbox (
  id             BIGSERIAL PRIMARY KEY,
  aggregate_id   TEXT NOT NULL,  -- e.g., entry_id or context_id
  op             TEXT NOT NULL,  -- upsert_entry | delete_entry | upsert_context | delete_context
  payload        JSONB NOT NULL,
  status         TEXT NOT NULL DEFAULT 'pending',
  attempt_count  INT NOT NULL DEFAULT 0,
  leased_until   TIMESTAMPTZ,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS outbox_ready_idx ON outbox(status, next_attempt_at);
```

Conventions:
- DB-generated timestamps via `DEFAULT now()`; app must not send creation times.
- Updates should set `last_update_time = now()` in DML.

## Adapter & Config Changes
- Add `internal/storage/postgres/adapter.go` using `pgx/v5` + `database/sql`. [Added]
- Implement all Storage methods with SQL shown above. [Completed]
- Use `RETURNING` to fetch created rows/creation_time.
- Add config keys: [Added]
  - `MEMORY_BACKEND_DB_DRIVER=postgres`
  - `MEMORY_BACKEND_POSTGRES_DSN` (e.g., `postgres://user:pass@host:5432/db?sslmode=disable`)
- Update factory (`internal/platform/factory/storage.go`) to construct Postgres adapter. [Added]

### Vault Isolation & Memory Operations
- **Strict vault_id scoping**: All memory, entry, and context operations require vault_id in WHERE clauses
- **Cross-vault operations**: Memory move (AddMemoryToVault) allowed; cross-vault sharing forbidden
- **Memory move endpoint**: `POST /api/users/{userId}/vaults/{targetVaultId}/memories/{memoryId}/attach`
- **Hard delete propagation**: DeleteMemoryFromVault delegates to existing hard delete flow with outbox events

## Outbox Worker
- On write/delete, insert outbox row in the same DB transaction.
- Worker loop (in a separate service/container):
  - `SELECT id, op, payload, aggregate_id FROM outbox WHERE status='pending' AND next_attempt_at <= now() FOR UPDATE SKIP LOCKED LIMIT 100`.
  - Perform idempotent Weaviate operation (upsert/delete) keyed by `entry_id`/`context_id` and tenant.
  - On success: `status='done'` (or delete row). On failure: increment `attempt_count`, set `next_attempt_at=now()+backoff`.

## Local Dev (Docker Compose)
- Add `postgres:16-alpine` service with volume and healthcheck. [Added as `deployments/docker/docker-compose.postgres.yml`]
- Provide a simple migration init container (`psql -f schema.sql`). [Added]
- Wire `memory-service` env to Postgres DSN; remove SQLite volume for new mode. [Added]
- Keep Weaviate and indexer as-is (indexer pointed at Postgres DSN). [Added]

## CI Changes
- Use Postgres service in CI for server tests.
- `make test-all` can keep SQLite path initially; add `backend-postgres-up` parallel target to switch when ready.
- Retire Spanner jobs and emulator setup from CI.

## Production (Aurora PG Serverless v2)
- Provision via Terraform: cluster, SGs, Secrets Manager creds, parameter group.
- App DSN from Secrets Manager; SSL required.
- Set max connections sensibly (Aurora v2 scales; still configure pool sizes in app).
- Optional pgbouncer not required initially.

## Risks & Mitigations
- Timestamp semantics differ from Spanner commit timestamps: use transaction `now()`; acceptable for ordering and API semantics.
- JSON marshaling differences: test round-trips for `context`, `metadata`, `tags`.
- Locking: ensure worker uses `FOR UPDATE SKIP LOCKED` to avoid contention.
- Migration cutover: dual-write window (optional) or maintenance window to export/import.

## Rollout Plan
1) Land Postgres adapter behind `DB_DRIVER=postgres` flag; keep existing paths. [Completed]
2) Add docker-compose Postgres and migration, add CI job to run server tests on Postgres. [Completed]
3) Switch local default to Postgres; deprecate SQLite path. [Completed]
4) Migrate dev/staging to Aurora PG; validate outbox + worker. [Pending]
5) Remove Spanner codepaths and emulator from repo once stabilized. [Pending]

## Implementation Status (2025-01-20)
- ✅ Postgres adapter with all Storage interface methods
- ✅ Vault isolation enforcement (vault_id scoping in all operations)
- ✅ Hard delete only (no expiration_time TTL)
- ✅ AddMemoryToVault/DeleteMemoryFromVault operations
- ✅ HTTP endpoint for memory move operations  
- ✅ Docker compose Postgres stack (removed in-stack Ollama dependency)
- ✅ Outbox worker integration with Weaviate indexing
- ✅ Full test suite passing (`make test-all`)

## Developer Notes
- Prefer `TIMESTAMPTZ` + `DEFAULT now()`; fetch times via `RETURNING`.
- Use prepared statements; scan JSONB into `[]byte` then unmarshal.
- Keep `entry_id` unique index for O(1) lookups and idempotency.

