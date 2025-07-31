---
title: Relational Storage Abstraction & PostgreSQL Dialect Alignment
status: Proposed
date: 2025-07-23
supersedes: "0002-database-strategy.md"
---

# Context

ADR-0002 committed to Cloud Spanner (GoogleSQL) as the sole datastore with an emulator for local dev. The new multi-target strategy requires additional engines but we want **one SQL dialect** to avoid query forks.

Cloud Spanner now offers a **PostgreSQL interface** that is 80-90 % compatible with standard PG. SQLite (modernc) also supports most PG syntax. Therefore adopting the PostgreSQL dialect unlocks:

* Spanner-PG on GCP (cloud & cloud-dev)
* Vanilla PostgreSQL / Aurora-PG in other clouds
* SQLite for local persistence

# Decision

1. Keep `internal/storage.Storage` as the service contract.
2. Adopt PostgreSQL-compatible SQL for all queries and DDL (schema.sql updated separately).
3. Implement adapters:
   * `spannerpgStorage` – uses Spanner PG interface via `cloud.google.com/go/spanner` (PG driver).
   * `postgresStorage` – uses `github.com/jackc/pgx/v5`.
   * `sqliteStorage` – uses `modernc.org/sqlite` (pure Go).
4. Factories select the adapter via `DB_DRIVER` (`spanner-pg`, `postgres`, `sqlite`).
5. CI runs invariant suite against `spanner-pg` and `sqlite` drivers.

# Consequences

• No SQL fork per engine; future migrations run on all targets.
• Spanner Emulator remains viable (PG mode).
• Minor loss of Spanner-specific features (commit timestamps) mitigated with explicit columns.

# Supersedes

ADR-0002 – Use Cloud Spanner with Emulator for Local Development (Spanner remains supported but the “sole datastore” stance is obsolete).

# References

* ADR-0015 Multi-Target Build Strategy. 