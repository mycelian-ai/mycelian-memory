---
title: Remove Spanner and SQLite; Postgres-only backend
status: Accepted
date: 2025-08-09
---

## Context

ADR 0014 proposed migrating storage from Spanner/SQLite to PostgreSQL locally and Aurora PostgreSQL Serverless v2 in production. That migration is complete and validated across the server test suite. Maintaining Spanner and SQLite paths creates code drift, higher CI costs, and complicates operational runbooks.

## Decision

- Remove Spanner and SQLite codepaths and tooling from the repository.
- Make PostgreSQL the only supported storage backend for all environments.

## Scope

- Delete Spanner adapter, tests, and GoogleSQL schema files.
- Delete SQLite adapter and local helpers.
- Remove Spanner and SQLite docker-compose stacks and emulator setup scripts.
- Simplify configuration: default and only supported `DB_DRIVER=postgres`; remove Spanner/SQLite fields and helpers.
- Update factory to error on unsupported drivers.
- Adjust tests to run on Postgres only; skip/remove Spanner harness-dependent tests.

## Consequences

- Reduced complexity in code, tests, and ops.
- Single SQL dialect (Postgres) simplifies development and maintenance.
- Any future multi-backend support will require new ADR(s) and isolated modules.

## Implementation Summary

- Server
  - Removed `internal/storage/spanner.go`, `spanner_*_test.go`, `internal/storage/schema.sql`.
  - Removed `internal/storage/sqlite/*` and local schema helpers; scrubbed factory case.
  - Updated `internal/config/config.go` to Postgres-only; removed Spanner/SQLite config.
  - Updated `internal/factory/storage.go` to return an error for `spanner-pg`.
  - Skipped/removed API tests that depended on Spanner emulator harness.
  - Kept search/outbox unchanged (Postgres source of truth; Waviate indexing intact).
- Tooling & Compose
  - Deleted `deployments/docker/docker-compose.spanner.yml` and SQLite compose stack.
  - Deleted `server/scripts/docker-setup/*` (Spanner emulator setup/health).
  - Simplified Makefiles to Postgres-only targets; removed emulator/schema targets.

## Rollback

Reintroducing Spanner/SQLite would require new adapters, config keys, test harnesses, and compose stacks; this ADR documents the removal and rationale.

## Status

- Code landed. Full server test suite green on Postgres.


